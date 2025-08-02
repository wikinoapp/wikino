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
    space = SpaceRepository.new.to_model(space_record: attachment_record.space_record.not_nil!)
    attached_space_member = SpaceMemberRepository.new.to_model(
      space_member_record: attachment_record.attached_space_member_record.not_nil!
    )

    Attachment.new(
      database_id: attachment_record.id,
      space:,
      filename: attachment_record.filename.not_nil!,
      content_type: attachment_record.content_type.not_nil!,
      byte_size: attachment_record.byte_size.not_nil!,
      attached_space_member:,
      attached_at: attachment_record.attached_at,
      url:
    )
  end

  sig do
    params(
      attachment_records: T.any(
        AttachmentRecord::PrivateCollectionProxy,
        AttachmentRecord::PrivateAssociationRelation,
        ActiveRecord::Relation
      ),
      space_member_record: T.nilable(SpaceMemberRecord),
      include_urls: T::Boolean
    ).returns(T::Array[Attachment])
  end
  def to_models(attachment_records:, space_member_record: nil, include_urls: false)
    attachment_records.map do |attachment_record|
      url = if include_urls
        attachment_record.generate_signed_url(space_member_record:)
      end
      to_model(attachment_record: attachment_record, url: url)
    end
  end
end
