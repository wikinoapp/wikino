# typed: strict
# frozen_string_literal: true

class LinkRepository < ApplicationRepository
  sig do
    params(
      page_records: T::Array[PageRecord],
      added_page_ids: T::Array[T::Wikino::DatabaseId],
      backlink_limit: Integer,
      user_record: T.nilable(UserRecord)
    ).returns(T::Array[Link])
  end
  def to_models(page_records:, added_page_ids:, backlink_limit:, user_record:)
    page_records.map do |page_record|
      added_page_ids << page_record.id

      cursor_paginate_page = page_record.backlinked_page_records(user_record:)
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
          page: PageRepository.new.to_model(page_record: backlinked_page_record, current_space_member: nil)
        )
      end

      backlink_list = BacklinkList.new(
        backlinks:,
        pagination: PaginationRepository.new.to_model(cursor_paginate_page:)
      )

      Link.new(
        page: PageRepository.new.to_model(page_record:, current_space_member: nil),
        backlink_list:
      )
    end
  end
end
