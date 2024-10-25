# typed: strict
# frozen_string_literal: true

class Page < ApplicationRecord
  include ModelConcerns::Pageable

  acts_as_sequenced column: :number, scope: :space_id

  belongs_to :topic
  belongs_to :space
  has_many :editorships, class_name: "PageEditorship", dependent: :restrict_with_exception
  has_many :revisions, class_name: "PageRevision", dependent: :restrict_with_exception

  scope :published, -> { where.not(published_at: nil).where(archived_at: nil) }
  scope :initial, -> { where(title: nil) }

  # validates :body, length: {maximum: 1_000_000}
  # validates :original, absence: true

  # sig { returns(T.nilable(Page)) }
  # def original
  #   user&.pages_except(self)&.find_by(title:)
  # end

  sig { params(current_space: Space, page_locations: T::Array[PageLocation]).returns(Page::PrivateRelation) }
  def self.all_from_page_locations(current_space:, page_locations:)
    page_ids = page_locations.group_by(&:topic_name).each_with_object([]) do |(topic_name, locations), ary|
      ary.concat(current_space.pages.joins(:topic).where(topics: {name: topic_name}, title: locations.map(&:page_title)).pluck(:id))
    end
    where(id: page_ids)
  end

  sig { params(topic: Topic).returns(Page) }
  def self.create_as_initial!(topic:)
    initial.where(topic:).first_or_create!(
      space: topic.space,
      title: nil,
      body: "",
      body_html: "",
      linked_page_ids: [],
      modified_at: Time.zone.now
    )
  end

  T::Sig::WithoutRuntime.sig { returns(Page::PrivateRelation) }
  def backlinked_pages
    Page.where("'#{id}' = ANY (linked_page_ids)")
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
