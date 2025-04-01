# typed: strict
# frozen_string_literal: true

class Export < ApplicationRecord
  has_one_attached :file

  belongs_to :space
  belongs_to :started_by, class_name: "SpaceMember"
  has_many :export_logs, dependent: :restrict_with_exception

  sig { returns(T::Boolean) }
  def finished?
    finished_at.present?
  end

  sig { params(message_key: Symbol, logged_at: ActiveSupport::TimeWithZone).void }
  def add_log!(message_key:, logged_at:)
    export_logs.create!(
      space:,
      message: I18n.t("messages.export_logs.#{message_key}"),
      logged_at:
    )
  end
end
