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

  scope :public_or_private, -> { where(visibility: [TopicVisibility::Public.serialize, TopicVisibility::Private.serialize]) }

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(TopicEntity) }
  def to_entity(space_viewer:)
    TopicEntity.new(
      database_id: id,
      number:,
      name:,
      description:,
      visibility: TopicVisibility.deserialize(visibility),
      space_entity: space_record.not_nil!.to_entity(space_viewer:),
      viewer_can_create_page: space_viewer.can_create_page?(topic: self),
      viewer_can_update: space_viewer.can_update_topic?(topic: self)
    )
  end

  sig { params(member: SpaceMember, role: TopicMemberRole, joined_at: ActiveSupport::TimeWithZone).void }
  def add_member!(member:, role:, joined_at: Time.zone.now)
    members.create!(space: member.space, space_member: member, role: role.serialize, joined_at:)

    nil
  end
end
