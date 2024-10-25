# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module Pageable
    include Kernel

    extend ActiveSupport::Concern
    extend T::Sig

    sig { returns(Page) }
    def original_page
      (instance_of?(DraftPage) ? T.bind(self, DraftPage).page : T.bind(self, Page)).not_nil!
    end

    T::Sig::WithoutRuntime.sig { returns(Page::PrivateRelation) }
    def linked_pages
      Page.where(id: linked_page_ids)
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

    sig { params(editor: User).void }
    def link!(editor:)
      location_keys = PageLocationKey.scan_text(text: body, current_topic: topic)
      topics = Topic.where(name: location_keys.map(&:topic_name))

      linked_pages = location_keys.each_with_object([]) do |location_key, ary|
        page_topic = topics.find { |topic| topic.name == location_key.topic_name }

        if page_topic
          ary << editor.create_linked_page!(topic: page_topic, title: location_key.page_title)
        end
      end

      update!(linked_page_ids: linked_pages.pluck(:id))

      nil
    end
  end
end
