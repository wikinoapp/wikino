# typed: strict
# frozen_string_literal: true

class Export < ApplicationRecord
  has_one_attached :file

  belongs_to :space
  belongs_to :queued_by, class_name: "SpaceMember"
  has_many :statuses, class_name: "ExportStatus", dependent: :restrict_with_exception
  has_many :logs, class_name: "ExportLog", dependent: :restrict_with_exception
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

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(ExportEntity) }
  def to_entity(space_viewer:)
    ExportEntity.new(
      database_id: id,
      queued_by_entity: queued_by.not_nil!.to_entity(space_viewer:),
      space_entity: space.not_nil!.to_entity(space_viewer:)
    )
  end

  def target_pages
    space.pages.active
  end

  sig { params(kind: ExportStatusKind).void }
  def change_status!(kind:)
    statuses.create!(
      space:,
      kind: kind.serialize,
      changed_at: Time.current
    )
  end

  sig { params(message: String, logged_at: ActiveSupport::TimeWithZone).void }
  def add_log!(message:, logged_at: Time.current)
    logs.create!(space:, message:, logged_at:)
  end
end
