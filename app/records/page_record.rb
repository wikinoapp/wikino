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
  scope :active, -> { kept.not_trashed.published }
  scope :restorable, -> { where(trashed_at: Page::DELETE_LIMIT_DAYS.days.ago..) }

  sig do
    params(
      space_viewer: ModelConcerns::SpaceViewable,
      pages: T.any(PageRecord::PrivateAssociationRelation, T::Array[PageRecord])
    ).returns(T::Array[PageEntity])
  end
  def self.to_entities(space_viewer:, pages:)
    pages.map do |page|
      page.to_entity(space_viewer:)
    end
  end

  sig { params(topic: TopicRecord).returns(PageRecord) }
  def self.create_as_blanked!(topic:)
    topic.page_records.create!(
      space_record: topic.space_record,
      title: nil,
      body: "",
      body_html: "",
      linked_page_ids: [],
      modified_at: Time.current
    )
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

  T::Sig::WithoutRuntime.sig { returns(T.any(PageRecord::PrivateAssociationRelationWhereChain, PageRecord::PrivateAssociationRelation)) }
  def backlinked_page_records
    pages = space_record.not_nil!.page_records.where("'#{id}' = ANY (linked_page_ids)")

    pages.joins(:topic_record).merge(Current.viewer!.viewable_topics)
  end

  sig { params(editor: SpaceMemberRecord).void }
  def add_editor!(editor:)
    page_editor_records.where(space_record:, space_member_record: editor).first_or_create!(
      last_page_modified_at: modified_at
    )

    nil
  end

  sig { params(editor: SpaceMemberRecord, body: String, body_html: String).returns(PageRevisionRecord) }
  def create_revision!(editor:, body:, body_html:)
    revision_records.create!(space_record:, space_member_record: editor, body:, body_html:)
  end
end
