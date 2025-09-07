# typed: strict
# frozen_string_literal: true

class AttachmentProcessingJob < ApplicationJob
  queue_as :default

  # ファイルアップロード後の処理を非同期で実行する
  sig { params(attachment_record_id: Types::DatabaseId).void }
  def perform(attachment_record_id)
    attachment_record = AttachmentRecord.find_by(id: attachment_record_id)
    return unless attachment_record

    result = Attachments::ProcessService.new.call(attachment_record:)

    # 処理に失敗した場合（ファイルがまだアップロードされていない等）、リトライする
    unless result.success
      # pendingステータスの場合のみリトライ（failedの場合はリトライしない）
      if attachment_record.processing_status_pending?
        # 少し待ってからリトライ
        AttachmentProcessingJob.set(wait: 5.seconds).perform_later(attachment_record_id)
      end
    end
  end
end
