# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module PageEditable
    extend ActiveSupport::Concern
    extend T::Sig

    sig { returns(Page) }
    def original_page
      instance_of?(Page) ? self : page
    end

    sig { returns(T::Array[PageLocation]) }
    def paths_in_body
      current_topic_name = topic.name
      titles_with_topic = body.scan(%r{\[\[(.*?)\]\]}).flatten.map(&:strip)

      titles_with_topic.each_with_object([]) do |title_with_topic, ary|
        topic_name, page_title = title_with_topic.split("/", 2)

        if !topic_name.nil? && !page_title.nil?
          ary << PageLocation.new(topic_name:, page_title:)
        elsif !topic_name.nil? && page_title.nil?
          ary << PageLocation.new(topic_name: current_topic_name, page_title: topic_name)
        else
          next
        end
      end
    end

    T::Sig::WithoutRuntime.sig { returns(Page::PrivateRelation) }
    def linked_pages
      Page.where(id: linked_page_ids)
    end

    T::Sig::WithoutRuntime.sig { returns(Page::PrivateRelation) }
    def backlinked_pages
      Page.where("'#{id}' = ANY (linked_page_ids)")
    end

    sig do
      params(
        before: T.nilable(String),
        after: T.nilable(String),
        link_limit: Integer,
        backlink_limit: Integer
      ).returns(LinkCollection)
    end
    def fetch_link_collection(before: nil, after: nil, link_limit: 15, backlink_limit: 14)
      added_page_ids = [id]

      cursor_paginate_page = linked_pages.where.not(id: added_page_ids).preload(:topic).cursor_paginate(
        after:,
        before:,
        limit: link_limit,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      pages = cursor_paginate_page.records
      pagination = Pagination.from_cursor_paginate(cursor_paginate_page:)

      links = pages.map do |page|
        added_page_ids << page.id

        cursor_paginate_page = page.backlinked_pages.where.not(id: added_page_ids).preload(:topic).cursor_paginate(
          after:,
          before:,
          limit: backlink_limit,
          order: {modified_at: :desc, id: :desc}
        ).fetch
        backlinked_pages = cursor_paginate_page.records
        backlink_collection = BacklinkCollection.new(
          page:,
          backlinks: backlinked_pages.map { |backlinked_page| Backlink.new(page: backlinked_page) },
          pagination: Pagination.from_cursor_paginate(cursor_paginate_page:)
        )

        added_page_ids.concat(backlinked_pages.pluck(:id))

        Link.new(page:, backlink_collection:)
      end

      LinkCollection.new(page: original_page, links:, pagination:)
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
    def link!(editor:)
      topics = Topic.where(name: paths_in_body.map(&:topic_name))

      linked_pages = paths_in_body.each_with_object([]) do |path, ary|
        page_topic = topics.find { |topic| topic.name == path.topic_name }

        if page_topic
          ary << editor.create_linked_page!(topic: page_topic, title: path.page_title)
        end
      end

      update!(linked_page_ids: linked_pages.pluck(:id))

      nil
    end
  end
end
