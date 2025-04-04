# typed: strict
# frozen_string_literal: true

class Export < ApplicationRecord
  has_one_attached :file

  belongs_to :space
  belongs_to :started_by, class_name: "SpaceMember"
  has_many :logs, class_name: "ExportLog", dependent: :restrict_with_exception

  sig { returns(T::Boolean) }
  def finished?
    finished_at.present?
  end

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(ExportEntity) }
  def to_entity(space_viewer:)
    ExportEntity.new(
      database_id: id,
      started_by_entity: started_by.not_nil!.to_entity(space_viewer:),
      started_at:,
      finished_at:,
      space_entity: space.not_nil!.to_entity(space_viewer:)
    )
  end

  sig { params(message_key: Symbol, logged_at: ActiveSupport::TimeWithZone).void }
  def add_log!(message_key:, logged_at:)
    logs.create!(
      space:,
      message: I18n.t("messages.export_logs.#{message_key}"),
      logged_at:
    )
  end
end
