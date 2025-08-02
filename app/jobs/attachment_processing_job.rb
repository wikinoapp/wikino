# typed: strict
# frozen_string_literal: true

class AttachmentProcessingJob < ApplicationJob
  queue_as :default

  # ファイルアップロード後の処理を非同期で実行する
  sig { params(attachment_record_id: T::Wikino::DatabaseId).void }
  def perform(attachment_record_id)
    attachment_record = AttachmentRecord.find_by(id: attachment_record_id)
    return unless attachment_record

    service = Attachments::ProcessService.new
    service.call(attachment_record:)
  end
end
