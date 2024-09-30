# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module PageEditable
    extend ActiveSupport::Concern
    extend T::Sig

    sig { returns(T::Array[String]) }
    def titles_in_body
      body.scan(%r{\[\[(.*?)\]\]}).flatten.map(&:strip)
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
