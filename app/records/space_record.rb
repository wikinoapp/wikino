# typed: strict
# frozen_string_literal: true

class SpaceRecord < ApplicationRecord
  include Discard::Model

  include FormConcerns::ISpace

  self.table_name = "spaces"

  enum :plan, {
    Plan::Free.serialize => 0,
    Plan::Small.serialize => 1,
    Plan::Large.serialize => 2
  }, prefix: true

  has_many :draft_page_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :export_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :topic_member_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :topic_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :page_editor_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :page_revision_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :page_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record
  has_many :space_member_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_id,
    inverse_of: :space_record

  scope :visible, -> { kept }

  sig do
    params(identifier: String, current_time: ActiveSupport::TimeWithZone, locale: Locale)
      .returns(SpaceRecord)
  end
  def self.create_initial_space!(identifier:, current_time:, locale:)
    create!(
      identifier:,
      plan: Plan::Small.serialize,
      name: I18n.t("messages.spaces.new_space", locale: locale.serialize),
      joined_at: current_time
    )
  end

  sig { params(identifier: String).returns(SpaceRecord) }
  def self.find_by_identifier!(identifier)
    kept.find_by!(identifier:)
  end

  sig { override.params(identifier: String).returns(T::Boolean) }
  def identifier_uniqueness?(identifier)
    SpaceRecord.where.not(id:).exists?(identifier:)
  end

  sig { params(number: Integer).returns(PageRecord) }
  def find_page_by_number!(number)
    page_records.visible.find_by!(number:)
  end

  sig do
    params(
      user_record: UserRecord,
      role: SpaceMemberRole,
      joined_at: ActiveSupport::TimeWithZone
    ).void
  end
  def add_member!(user_record:, role:, joined_at:)
    space_member_records.create!(user_record:, role: role.serialize, joined_at:)
  end
end
