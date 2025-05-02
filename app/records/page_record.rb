# typed: strict
# frozen_string_literal: true

class PageRecord < ApplicationRecord
  include Discard::Model

  include RecordConcerns::Pageable

  self.table_name = "pages"

  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :topic_record, foreign_key: :topic_id
  belongs_to :space_record, foreign_key: :space_id
  has_many :page_editor_records,
    class_name: "PageEditorRecord",
    dependent: :restrict_with_exception,
    foreign_key: :page_id,
    inverse_of: :page_record
  has_many :revision_records,
    class_name: "PageRevisionRecord",
    dependent: :restrict_with_exception,
    foreign_key: :page_id,
    inverse_of: :page_record

  scope :published, -> { where.not(published_at: nil) }
  scope :pinned, -> { where.not(pinned_at: nil) }
  scope :not_pinned, -> { where(pinned_at: nil) }
  scope :not_trashed, -> { where(trashed_at: nil) }
  scope :topics_kept, -> { joins(:topic_record).merge(TopicRecord.kept) }
  scope :topics_visibility_public, -> { joins(:topic_record).merge(TopicRecord.visibility_public) }
  scope :visible, -> { kept.not_trashed.topics_kept }
  scope :active, -> { visible.published }
  scope :restorable, -> { where(trashed_at: Page::DELETE_LIMIT_DAYS.days.ago..) }

  sig { params(topic_record: TopicRecord).returns(PageRecord) }
  def self.create_as_blanked!(topic_record:)
    topic_record.page_records.create!(
      space_record: topic_record.space_record,
      title: nil,
      body: "",
      body_html: "",
      linked_page_ids: [],
      modified_at: Time.current
    )
  end

  sig { returns(SpaceRecord) }
  def space_record!
    space_record.not_nil!
  end

  sig { returns(TopicRecord) }
  def topic_record!
    topic_record.not_nil!
  end

  sig { returns(T::Boolean) }
  def pinned?
    pinned_at.present?
  end

  sig { returns(T::Boolean) }
  def published?
    published_at.present?
  end

  sig { returns(T::Boolean) }
  def trashed?
    trashed_at.present?
  end

  sig do
    params(
      user_record: T.nilable(UserRecord)
    ).returns(
      T.any(
        PageRecord::PrivateAssociationRelationWhereChain,
        PageRecord::PrivateAssociationRelation
      )
    )
  end
  def backlinked_page_records(user_record:)
    pages = space_record.not_nil!.page_records.visible.where("'#{id}' = ANY (linked_page_ids)")
    topic_records = user_record.nil? ? TopicRecord.visibility_public : user_record.viewable_topics

    pages.joins(:topic_record).merge(topic_records)
  end

  sig { params(editor_record: SpaceMemberRecord).void }
  def add_editor!(editor_record:)
    page_editor_records.where(space_record:, space_member_record: editor_record).first_or_create!(
      last_page_modified_at: modified_at
    )

    nil
  end

  sig do
    params(
      editor_record: SpaceMemberRecord,
      body: String,
      body_html: String
    ).returns(PageRevisionRecord)
  end
  def create_revision!(editor_record:, body:, body_html:)
    revision_records.create!(space_record:, space_member_record: editor_record, body:, body_html:)
  end
end
