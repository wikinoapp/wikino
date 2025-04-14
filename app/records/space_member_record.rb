# typed: strict
# frozen_string_literal: true

class SpaceMemberRecord < ApplicationRecord
  include ModelConcerns::SpaceViewable

  self.table_name = "space_members"

  enum :role, {
    SpaceMemberRole::Owner.serialize => 0
  }, prefix: true

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :user_record, foreign_key: :user_id
  has_many :topic_member_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_member_id,
    inverse_of: :space_member_record
  has_many :topic_records,
    foreign_key: :topic_id,
    through: :topic_member_records
  has_many :draft_page_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_member_id,
    inverse_of: :space_member_record
  has_many :page_editor_records,
    dependent: :restrict_with_exception,
    foreign_key: :space_member_id,
    inverse_of: :space_member_record

  delegate :locale, :time_zone, to: :user, prefix: true

  scope :active, -> { where(active: true) }
  scope :inactive, -> { where(active: false) }

  sig { params(topic: Topic, title: String).returns(Page) }
  def create_linked_page!(topic:, title:)
    page = space.not_nil!.pages.where(topic:, title:).first_or_create!(
      space:,
      body: "",
      body_html: "",
      linked_page_ids: [],
      modified_at: Time.zone.now
    )
    page_editors.where(page:).first_or_create!(space:, last_page_modified_at: page.modified_at)

    page
  end

  sig { params(page: PageRecord).returns(DraftPageRecord) }
  def find_or_create_draft_page!(page:)
    draft_pages.create_with(
      space: page.space,
      topic: page.topic,
      title: page.title,
      body: page.body,
      body_html: page.body_html,
      linked_page_ids: page.linked_page_ids,
      modified_at: Time.zone.now
    ).find_or_create_by!(page:)
  rescue ActiveRecord::RecordNotUnique
    retry
  end

  sig { params(page: PageRecord).void }
  def destroy_draft_page!(page:)
    draft_pages.where(page:).destroy_all

    nil
  end

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(SpaceMemberEntity) }
  def to_entity(space_viewer:)
    SpaceMemberEntity.new(
      database_id: id,
      space_entity: space.not_nil!.to_entity(space_viewer:),
      user_entity: user.not_nil!.to_entity
    )
  end

  sig { returns(SpaceMemberRole) }
  def deserialized_role
    SpaceMemberRole.deserialize(role)
  end

  sig { returns(T::Array[SpaceMemberPermission]) }
  def permissions
    deserialized_role.permissions
  end

  sig { returns(Page::PrivateAssociationRelation) }
  def last_modified_pages
    space.not_nil!.pages.joins(:editors).merge(
      page_editors.order(PageEditor.arel_table[:last_page_modified_at].desc)
    )
  end

  sig { override.returns(Page::PrivateAssociationRelation) }
  def showable_pages
    space.not_nil!.pages.active
  end

  sig { override.returns(T.any(Topic::PrivateAssociationRelation, Topic::PrivateRelation)) }
  def joined_topics
    topics.kept
  end

  sig { override.returns(Topic::PrivateAssociationRelation) }
  def showable_topics
    space.not_nil!.topics.kept
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_update_space?(space:)
    space.id == space_id && permissions.include?(SpaceMemberPermission::UpdateSpace)
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_export_space?(space:)
    space.id == space_id && permissions.include?(SpaceMemberPermission::ExportSpace)
  end

  sig { override.params(topic: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic:)
    space.not_nil!.id == topic.space_id && permissions.include?(SpaceMemberPermission::UpdateTopicRecord)
  end

  sig { override.params(topic: T.nilable(TopicRecord)).returns(T::Boolean) }
  def can_create_page?(topic:)
    topic.present? && topics.where(id: topic.id).exists?
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_create_bulk_restored_pages?(space:)
    active? && space_id == space.id
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_view_page?(page:)
    active? && space_id == page.space_id
  end

  sig { override.params(space: Space).returns(T::Boolean) }
  def can_view_trash?(space:)
    active? && space_id == space.id
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_update_page?(page:)
    active? && joined_topics.where(id: page.topic_id).exists?
  end

  sig { override.returns(T::Boolean) }
  def can_create_topic?
    true
  end

  sig { override.params(page: Page).returns(T::Boolean) }
  def can_update_draft_page?(page:)
    topics.where(id: page.topic_id).exists?
  end
end
