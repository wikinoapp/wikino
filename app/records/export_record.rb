# typed: strict
# frozen_string_literal: true

class Export < ApplicationRecord
  PRESIGNED_URL_EXPIRATION = 86400 # 24時間

  has_one_attached :file

  belongs_to :space
  belongs_to :queued_by, class_name: "SpaceMember"
  has_many :statuses, class_name: "ExportStatus", dependent: :restrict_with_exception
  has_one :latest_status, -> { order(changed_at: :desc) }, class_name: "ExportStatus", inverse_of: false

  sig { returns(ExportStatusKind) }
  def latest_status_kind
    ExportStatusKind.deserialize(latest_status.not_nil!.kind)
  end

  sig { returns(T::Boolean) }
  def failed?
    latest_status_kind == ExportStatusKind::Failed
  end

  sig { returns(T::Boolean) }
  def succeeded?
    latest_status_kind == ExportStatusKind::Succeeded
  end

  sig { returns(T::Boolean) }
  def active?
    succeeded? && latest_status.not_nil!.changed_at > 1.day.ago
  end

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(ExportEntity) }
  def to_entity(space_viewer:)
    ExportEntity.new(
      database_id: id,
      queued_by_entity: queued_by.not_nil!.to_entity(space_viewer:),
      space_entity: space.not_nil!.to_entity(space_viewer:)
    )
  end

  sig { returns(Page::PrivateAssociationRelation) }
  def target_pages
    space.not_nil!.pages.active
  end

  sig { returns(String) }
  def presigned_url
    signer = Aws::S3::Presigner.new(client: ActiveStorage::Blob.service.client.client)

    # アップロード用Presigned URLの生成
    signer.presigned_url(
      :get_object,
      bucket: ActiveStorage::Blob.service.bucket.name,
      key: file.key,
      expires_in: PRESIGNED_URL_EXPIRATION
    )
  end

  sig { params(kind: ExportStatusKind).void }
  def change_status!(kind:)
    statuses.create!(
      space:,
      kind: kind.serialize,
      changed_at: Time.current
    )
  end

  sig { void }
  def send_succeeded_mail!
    ExportMailer.succeeded(export_id: id, locale: queued_by.not_nil!.user_locale).deliver_later

    nil
  end
end
