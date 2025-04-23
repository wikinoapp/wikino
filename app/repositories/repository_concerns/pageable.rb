# typed: strict
# frozen_string_literal: true

module RepositoryConcerns
  module Pageable
    extend T::Sig

    sig do
      params(
        pageable_record: RecordConcerns::Pageable,
        before: T.nilable(String),
        after: T.nilable(String),
        link_limit: Integer,
        backlink_limit: Integer
      ).returns(LinkList)
    end
    def link_list(pageable_record:, before: nil, after: nil, link_limit: 15, backlink_limit: 14)
      added_page_ids = [pageable_record.id]

      cursor_paginate_page = pageable_record.linked_pages
        .where.not(id: added_page_ids).preload(:topic_record)
        .cursor_paginate(
          after:,
          before:,
          limit: link_limit,
          order: {modified_at: :desc, id: :desc}
        ).fetch
      page_records = cursor_paginate_page.records

      links = fetch_links(page_records:, added_page_ids:, backlink_limit:)
      pagination = Pagination.from_cursor_paginate(cursor_paginate_page:)

      LinkList.new(
        links:,
        pagination:
      )
    end

    sig do
      params(
        page_records: T::Array[PageRecord],
        added_page_ids: T::Array[T::Wikino::DatabaseId],
        backlink_limit: Integer
      ).returns(T::Array[Link])
    end
    private def fetch_links(page_records:, added_page_ids:, backlink_limit:)
      page_records.map do |page_record|
        added_page_ids << page_record.id

        cursor_paginate_page = page_record.backlinked_page_records
          .where.not(id: added_page_ids).preload(:topic_record)
          .cursor_paginate(
            after: nil,
            before: nil,
            limit: backlink_limit,
            order: {modified_at: :desc, id: :desc}
          ).fetch
        backlinked_page_records = cursor_paginate_page.records
        added_page_ids.concat(backlinked_page_records.pluck(:id))

        backlinks = backlinked_page_records.map do |backlinked_page_record|
          Backlink.new(
            page: PageRepository.new.to_model(page_record: backlinked_page_record)
          )
        end

        backlink_list = BacklinkList.new(
          backlinks:,
          pagination: Pagination.from_cursor_paginate(cursor_paginate_page:)
        )

        Link.new(
          page: PageRepository.new.to_model(page_record:),
          backlink_list:
        )
      end
    end
  end
end
