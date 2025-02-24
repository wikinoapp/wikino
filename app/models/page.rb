# typed: strict
# frozen_string_literal: true

class Page < ApplicationRecord
  include Discard::Model
  include ModelConcerns::Pageable

  # ページをゴミ箱に移動してから削除されるまでの日数
  DELETE_LIMIT_DAYS = 30

  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :topic
  belongs_to :space
  has_many :editors, class_name: "PageEditor", dependent: :restrict_with_exception
  has_many :revisions, class_name: "PageRevision", dependent: :restrict_with_exception

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
      PageEntity.new(
        database_id: page.id,
        number: page.number,
        title: page.title,
        body: page.body,
        body_html: page.body_html,
        modified_at: page.modified_at,
        published_at: page.published_at,
        pinned_at: page.pinned_at,
        space_entity: page.space.to_entity(space_viewer:),
        topic_entity: page.topic.to_entity(space_viewer:)
      )
    end
  end

  sig { params(topic: Topic).returns(Page) }
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

  sig { returns(T::Boolean) }
  def modified_after_published?
    published? && modified_at > published_at
  end

  T::Sig::WithoutRuntime.sig { returns(T.any(Page::PrivateAssociationRelationWhereChain, Page::PrivateAssociationRelation)) }
  def backlinked_pages
    pages = space.not_nil!.pages.where("'#{id}' = ANY (linked_page_ids)")

    pages.joins(:topic).merge(Current.viewer!.viewable_topics)
  end

  sig { params(before: T.nilable(String), after: T.nilable(String), limit: Integer).returns(BacklinkCollection) }
  def fetch_backlink_collection(before: nil, after: nil, limit: 15)
    cursor_paginate_page = backlinked_pages.cursor_paginate(
      after:,
      before:,
      limit:,
      order: {modified_at: :desc, id: :desc}
    ).fetch

    backlinks = cursor_paginate_page.records.map do |page|
      Backlink.new(page:)
    end

    BacklinkCollection.new(
      page: original_page,
      backlinks:,
      pagination: Pagination.from_cursor_paginate(cursor_paginate_page:)
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
