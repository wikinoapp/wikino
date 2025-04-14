# typed: strict
# frozen_string_literal: true

class PageRecord < ApplicationRecord
  include Discard::Model

  include RecordConcerns::Pageable

  self.table_name = "pages"

  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :topic_record, foreign_key: :topic_id
  belongs_to :space_record, foreign_key: :space_id
  has_many :editor_records,
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
  scope :restorable, -> { where(trashed_at: DELETE_LIMIT_DAYS.days.ago..) }

  sig do
    params(
      space_viewer: ModelConcerns::SpaceViewable,
      pages: T.any(Page::PrivateAssociationRelation, T::Array[Page])
    ).returns(T::Array[PageEntity])
  end
  def self.to_entities(space_viewer:, pages:)
    pages.map do |page|
      page.to_entity(space_viewer:)
    end
  end

  sig { params(topic: TopicRecord).returns(PageRecord) }
  def self.create_as_blanked!(topic:)
    topic.pages.create!(
      space: topic.space,
      title: nil,
      body: "",
      body_html: "",
      linked_page_ids: [],
      modified_at: Time.current
    )
  end

  sig { params(space_viewer: ModelConcerns::SpaceViewable).returns(PageEntity) }
  def to_entity(space_viewer:)
    PageEntity.new(
      database_id: id,
      number:,
      title:,
      body:,
      body_html:,
      modified_at:,
      published_at:,
      pinned_at:,
      space_entity: space.not_nil!.to_entity(space_viewer:),
      topic_entity: topic.not_nil!.to_entity(space_viewer:),
      viewer_can_update: space_viewer.can_update_page?(page: self)
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

  T::Sig::WithoutRuntime.sig { returns(T.any(Page::PrivateAssociationRelationWhereChain, Page::PrivateAssociationRelation)) }
  def backlinked_pages
    pages = space.not_nil!.pages.where("'#{id}' = ANY (linked_page_ids)")

    pages.joins(:topic).merge(Current.viewer!.viewable_topics)
  end

  sig do
    params(
      space_viewer: ModelConcerns::SpaceViewable,
      before: T.nilable(String),
      after: T.nilable(String),
      limit: Integer
    ).returns(BacklinkListEntity)
  end
  def fetch_backlink_list_entity(space_viewer:, before: nil, after: nil, limit: 15)
    cursor_paginate_page = backlinked_pages.cursor_paginate(
      after:,
      before:,
      limit:,
      order: {modified_at: :desc, id: :desc}
    ).fetch

    backlink_entities = cursor_paginate_page.records.map do |page|
      BacklinkEntity.new(page_entity: page.to_entity(space_viewer:))
    end

    BacklinkListEntity.new(
      backlink_entities:,
      pagination_entity: PaginationEntity.from_cursor_paginate(cursor_paginate_page:)
    )
  end

  sig { params(editor: SpaceMember).void }
  def add_editor!(editor:)
    editors.where(space:, space_member: editor).first_or_create!(
      last_page_modified_at: modified_at
    )

    nil
  end

  sig { params(editor: SpaceMember, body: String, body_html: String).returns(PageRevision) }
  def create_revision!(editor:, body:, body_html:)
    revisions.create!(space:, space_member: editor, body:, body_html:)
  end
end
