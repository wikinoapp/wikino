# typed: strict
# frozen_string_literal: true

class TopicRecord < ApplicationRecord
  include Discard::Model

  self.table_name = "topics"

  acts_as_sequenced column: :number, scope: :space_id

  enum :visibility, {
    TopicVisibility::Public.serialize => 0,
    TopicVisibility::Private.serialize => 1
  }, prefix: true

  belongs_to :space_record, foreign_key: :space_id
  has_many :member_records, class_name: "TopicMemberRecord", foreign_key: :topic_id, dependent: :restrict_with_exception
  has_many :page_records, dependent: :restrict_with_exception, foreign_key: :topic_id, inverse_of: :topic_record

  scope :public_visibility, -> { where(visibility: TopicVisibility::Public.serialize) }

  sig { params(member: SpaceMemberRecord, role: TopicMemberRole, joined_at: ActiveSupport::TimeWithZone).void }
  def add_member!(member:, role:, joined_at: Time.zone.now)
    member_records.create!(space_record: member.space_record, space_member_record: member, role: role.serialize, joined_at:)

    nil
  end
end
