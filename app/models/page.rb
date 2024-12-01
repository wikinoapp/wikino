# typed: strict
# frozen_string_literal: true

class Page < ApplicationRecord
  include ModelConcerns::Pageable

  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :topic
  belongs_to :space
  has_many :editorships, class_name: "PageEditorship", dependent: :restrict_with_exception
  has_many :revisions, class_name: "PageRevision", dependent: :restrict_with_exception

  scope :published, -> { where.not(published_at: nil) }
  scope :pinned, -> { where.not(pinned_at: nil) }
  scope :not_pinned, -> { where(pinned_at: nil) }
  scope :not_trashed, -> { where(trashed_at: nil) }
  scope :active, -> { not_trashed.published }

  # validates :body, length: {maximum: 1_000_000}
  # validates :original, absence: true

  # sig { returns(T.nilable(Page)) }
  # def original
  #   user&.pages_except(self)&.find_by(title:)
  # end

  sig { params(topic: Topic).returns(Page) }
  def self.create_as_blanked!(topic:)
    topic.pages.create!(
      space: topic.space,
      title: nil,
      body: "",
      body_html: "",
      linked_page_ids: []
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
  def modified_after_published?
    published? && modified_at > published_at
  end

  T::Sig::WithoutRuntime.sig { returns(T.any(Page::PrivateAssociationRelationWhereChain, Page::PrivateAssociationRelation)) }
  def backlinked_pages
    pages = space.not_nil!.pages.where("'#{id}' = ANY (linked_page_ids)")

    if Current.user
      pages
    else
      pages.joins(:topic).merge(Topic.visibility_public)
    end
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

  sig { params(editor: User).void }
  def add_editor!(editor:)
    editorships.where(space:, editor:).first_or_create!(
      last_page_modified_at: modified_at
    )

    nil
  end

  sig { params(editor: User, body: String, body_html: String).returns(PageRevision) }
  def create_revision!(editor:, body:, body_html:)
    revisions.create!(space:, editor:, body:, body_html:)
  end
end
