# typed: strict
# frozen_string_literal: true

class PageRepository < ApplicationRepository
  include RepositoryConcerns::Pageable

  sig { params(page_record: PageRecord).returns(Page) }
  def to_model(page_record:)
    Page.new(
      database_id: page_record.id,
      number: page_record.number,
      title: page_record.title,
      body: page_record.body,
      body_html: page_record.body_html,
      modified_at: page_record.modified_at,
      published_at: page_record.published_at,
      pinned_at: page_record.pinned_at,
      space: SpaceRepository.new.to_model(space_record: page_record.space_record.not_nil!),
      topic: TopicRepository.new.to_model(topic_record: page_record.topic_record.not_nil!)
    )
  end

  sig do
    params(
      user_record: T.nilable(UserRecord),
      page_record: PageRecord,
      before: T.nilable(String),
      after: T.nilable(String),
      limit: Integer
    ).returns(BacklinkList)
  end
  def backlink_list(user_record:, page_record:, before: nil, after: nil, limit: 15)
    cursor_paginate_page = page_record.backlinked_page_records(user_record:)
      .cursor_paginate(
        after:,
        before:,
        limit:,
        order: {modified_at: :desc, id: :desc}
      ).fetch

    backlinks = cursor_paginate_page.records.map do |page_record|
      Backlink.new(
        page: to_model(page_record:)
      )
    end

    BacklinkList.new(
      backlinks:,
      pagination: Pagination.from_cursor_paginate(cursor_paginate_page:)
    )
  end
end
