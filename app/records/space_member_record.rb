# typed: strict
# frozen_string_literal: true

class SpaceMemberRecord < ApplicationRecord
  self.table_name = "space_members"

  enum :role, {
    SpaceMemberRole::Owner.serialize => 0,
    SpaceMemberRole::Member.serialize => 1
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

  delegate :locale, :time_zone, to: :user_record, prefix: true

  scope :active, -> { where(active: true) }
  scope :inactive, -> { where(active: false) }

  sig { params(topic_record: TopicRecord, title: String).returns(PageRecord) }
  def create_linked_page!(topic_record:, title:)
    page = space_record.not_nil!.page_records.where(topic_record:, title:).first_or_create!(
      space_record:,
      body: "",
      body_html: "",
      linked_page_ids: [],
      modified_at: Time.zone.now
    )
    page_editor_records.where(page_record: page).first_or_create!(space_record:, last_page_modified_at: page.modified_at)

    page
  end

  sig { params(page: PageRecord).returns(DraftPageRecord) }
  def find_or_create_draft_page!(page:)
    draft_page_records.create_with(
      space_record: page.space_record,
      topic_record: page.topic_record,
      title: page.title,
      body: page.body,
      body_html: page.body_html,
      linked_page_ids: page.linked_page_ids,
      modified_at: Time.zone.now
    ).find_or_create_by!(page_record: page)
  rescue ActiveRecord::RecordNotUnique
    retry
  end

  sig { params(page_record: PageRecord).void }
  def destroy_draft_page!(page_record:)
    draft_page_records.where(page_record:).destroy_all

    nil
  end

  sig { returns(SpaceMemberRole) }
  def deserialized_role
    SpaceMemberRole.deserialize(role)
  end

  sig { returns(T::Array[SpaceMemberPermission]) }
  def permissions
    deserialized_role.permissions
  end

  sig { returns(PageRecord::PrivateAssociationRelation) }
  def last_modified_pages
    space_record.not_nil!.page_records.joins(:page_editor_records).merge(
      page_editor_records.order(PageEditorRecord.arel_table[:last_page_modified_at].desc)
    )
  end

  sig { returns(T.any(TopicRecord::PrivateAssociationRelation, TopicRecord::PrivateRelation)) }
  def joined_topic_records
    topic_records.kept
  end

  sig { params(space: SpaceRecord).returns(T::Boolean) }
  def can_create_bulk_restored_pages?(space:)
    active? && space_id == space.id
  end
end
