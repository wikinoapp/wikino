# typed: strict
# frozen_string_literal: true

class Export < ApplicationRecord
  has_one_attached :file

  belongs_to :space
  belongs_to :queued_by, class_name: "SpaceMember"
  has_many :logs, class_name: "ExportLog", dependent: :restrict_with_exception

  sig { returns(T::Boolean) }
  def finished?
    finished_at.present?
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

  sig { params(message: String, logged_at: ActiveSupport::TimeWithZone).void }
  def add_log!(message:, logged_at: Time.current)
    logs.create!(space:, message:, logged_at:)
  end
end
