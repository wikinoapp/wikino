# typed: strict
# frozen_string_literal: true

class AttachmentProcessingJob < ApplicationJob
  queue_as :default

  # ファイルアップロード後の処理を非同期で実行する
  sig { params(attachment_record_id: T::Wikino::DatabaseId).void }
  def perform(attachment_record_id)
    attachment_record = AttachmentRecord.find_by(id: attachment_record_id)
    return unless attachment_record

    # 処理が必要ない場合はスキップ
    return unless attachment_record.needs_processing?

    # 処理中としてマーク
    attachment_record.mark_as_processing!

    blob_record = attachment_record.blob_record
    return unless blob_record

    begin
      # SVGファイルの場合はサニタイズ処理を実行
      if blob_record.content_type == "image/svg+xml"
        unless attachment_record.sanitize_svg_content
          raise "SVG sanitization failed for attachment_record: #{attachment_record_id}"
        end
      end

      # 画像ファイルの場合はEXIF削除と自動回転を実行
      if blob_record.image?
        unless blob_record.process_image_with_exif_removal
          raise "Image processing failed for attachment_record: #{attachment_record_id}"
        end
      end

      # 処理完了としてマーク
      attachment_record.mark_as_completed!
    rescue => e
      Rails.logger.error("AttachmentProcessingJob failed: #{e.message}")
      Rails.logger.error(e.backtrace.join("\n"))

      # 処理失敗としてマーク
      attachment_record.mark_as_failed!

      # エラーを再発生させてジョブをリトライ可能にする
      raise
    end
  end
end
