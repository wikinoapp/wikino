# typed: strict
# frozen_string_literal: true

class AttachmentRepository < ApplicationRepository
  sig do
    params(
      attachment_record: AttachmentRecord,
      url: T.nilable(String)
    ).returns(Attachment)
  end
  def to_model(attachment_record:, url: nil)
    Attachment.new(
      id: attachment_record.id,
      space_id: attachment_record.space_id,
      filename: attachment_record.filename.not_nil!,
      content_type: attachment_record.content_type.not_nil!,
      byte_size: attachment_record.byte_size.not_nil!,
      attached_user_id: attachment_record.attached_user_id,
      attached_at: attachment_record.attached_at,
      url: url
    )
  end

  sig do
    params(
      attachment_records: T.any(
        AttachmentRecord::PrivateCollectionProxy,
        AttachmentRecord::PrivateAssociationRelation,
        ActiveRecord::Relation
      ),
      include_urls: T::Boolean
    ).returns(T::Array[Attachment])
  end
  def to_models(attachment_records:, include_urls: false)
    attachment_records.map do |attachment_record|
      url = if include_urls && attachment_record.blob_record
        # 署名付きURLを生成（1時間有効）
        attachment_record.blob_record.url(expires_in: 1.hour)
      end
      to_model(attachment_record: attachment_record, url: url)
    end
  end
end
