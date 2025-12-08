# typed: strict
# frozen_string_literal: true

class LinkListRepository < ApplicationRepository
  sig do
    params(
      user_record: T.nilable(UserRecord),
      pageable_record: RecordConcerns::Pageable,
      before: T.nilable(String),
      after: T.nilable(String),
      link_limit: Integer,
      backlink_limit: Integer
    ).returns(LinkList)
  end
  def to_model(user_record:, pageable_record:, before: nil, after: nil, link_limit: 15, backlink_limit: 14)
    added_page_ids = [pageable_record.id]

    cursor_paginate_page = pageable_record.linked_pages(user_record:)
      .where.not(id: added_page_ids).preload(:topic_record)
      .cursor_paginate(
        after:,
        before:,
        limit: link_limit,
        order: {modified_at: :desc, id: :desc}
      ).fetch
    page_records = cursor_paginate_page.records

    links = LinkRepository.new.to_models(page_records:, added_page_ids:, backlink_limit:, user_record:)
    pagination = PaginationRepository.new.to_model(cursor_paginate_page:)

    LinkList.new(
      links:,
      pagination:
    )
  end
end
