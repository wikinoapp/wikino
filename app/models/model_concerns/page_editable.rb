# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module PageEditable
    extend ActiveSupport::Concern
    extend T::Sig

    sig { returns(T::Array[PagePath]) }
    def paths_in_body
      raise StandardError("topic needs to be present") if topic.nil?

      titles_with_topic = body.scan(%r{\[\[(.*?)\]\]}).flatten.map(&:strip)
      topic_names = titles_with_topic.map do |title_with_topic|
        topic_name, page_title = title_with_topic.split("/", 2)
        page_title.nil? ? nil : topic_name
      end.compact
      topics = space.topics.where(name: topic_names)
      topics_with_name = topic_names.each_with_object({}) do |topic_name, hash|
        hash[topic_name] = topics.find { |topic| topic.name == topic_name }
      end
      current_topic_name = topic.name

      titles_with_topic.map do |title_with_topic|
        topic_name, page_title = title_with_topic.split("/", 2)

        if !topic_name.nil? && !page_title.nil? && topics_with_name[topic_name].nil?
          nil
        elsif !topic_name.nil? && page_title.nil?
          "#{current_topic_name}/#{topic_name}"
        else
          title_with_topic
        end
      end.compact
    end

    T::Sig::WithoutRuntime.sig { returns(Page::PrivateRelation) }
    def linked_pages
      Page.where(id: linked_page_ids)
    end

    T::Sig::WithoutRuntime.sig { returns(Page::PrivateRelation) }
    def backlinked_pages
      Page.where("'#{id}' = ANY (linked_page_ids)")
    end

    sig { params(before: T.nilable(String), after: T.nilable(String), limit: Integer).returns(LinkList) }
    def fetch_link_list(before: nil, after: nil, limit: 15)
      added_page_ids = [id]

      cursor_paginate_page = linked_pages.where.not(id: added_page_ids).cursor_paginate(
        after:,
        before:,
        limit:,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      pages = cursor_paginate_page.records
      pagination = Pagination.from_cursor_paginate(cursor_paginate_page:)

      links = pages.map do |page|
        added_page_ids << page.id

        cursor_paginate_page = page.backlinked_pages.where.not(id: added_page_ids).cursor_paginate(
          after:,
          before:,
          limit:,
          order: {modified_at: :desc, id: :desc}
        ).fetch
        backlinked_pages = cursor_paginate_page.records

        added_page_ids.concat(backlinked_pages.pluck(:id))

        Link.new(
          page:,
          backlinked_pages:,
          pagination: Pagination.from_cursor_paginate(cursor_paginate_page:)
        )
      end

      LinkList.new(links:, pagination:)
    end

    sig { params(before: T.nilable(String), after: T.nilable(String), limit: Integer).returns(BacklinkList) }
    def fetch_backlink_list(before: nil, after: nil, limit: 15)
      cursor_paginate_page = backlinked_pages.cursor_paginate(
        after:,
        before:,
        limit:,
        order: {modified_at: :desc, id: :desc}
      ).fetch

      backlinks = cursor_paginate_page.records.map do |page|
        Backlink.new(page:)
      end

      BacklinkList.new(backlinks:, pagination: Pagination.from_cursor_paginate(cursor_paginate_page:))
    end

    sig { params(editor: User).void }
    def link!(editor:)
      linked_pages = titles_in_body.map do |title|
        editor.create_linked_page!(topic: topic.not_nil!, title:)
      end

      update!(linked_page_ids: linked_pages.pluck(:id))
    end
  end
end
