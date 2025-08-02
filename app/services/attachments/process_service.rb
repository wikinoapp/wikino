# typed: strict
# frozen_string_literal: true

module Attachments
  class ProcessService < ApplicationService
    class Result < T::Struct
      const :attachment_record, AttachmentRecord
      const :success, T::Boolean
    end

    sig { params(attachment_record: AttachmentRecord).returns(Result) }
    def call(attachment_record:)
      # 処理が必要ない場合はスキップ
      unless attachment_record.processing_status_pending?
        return Result.new(attachment_record:, success: true)
      end

      # 処理中としてマーク
      attachment_record.processing_status_processing!

      blob_record = attachment_record.blob_record
      unless blob_record
        attachment_record.processing_status_failed!
        return Result.new(attachment_record:, success: false)
      end

      begin
        # SVGファイルの場合はサニタイズ処理を実行
        if blob_record.content_type == "image/svg+xml"
          unless attachment_record.sanitize_svg_content
            raise "SVG sanitization failed for attachment_record: #{attachment_record.id}"
          end
        end

        # 画像ファイルの場合はEXIF削除と自動回転を実行
        if blob_record.image?
          unless blob_record.process_image_with_exif_removal
            raise "Image processing failed for attachment_record: #{attachment_record.id}"
          end
        end

        # 処理完了としてマーク
        attachment_record.processing_status_completed!
        Result.new(attachment_record:, success: true)
      rescue => e
        Rails.logger.error("Attachments::ProcessService failed: #{e.message}")
        Rails.logger.error(e.backtrace.join("\n"))

        # 処理失敗としてマーク
        attachment_record.processing_status_failed!

        # エラーを再発生させる
        raise
      end
    end
  end
end
