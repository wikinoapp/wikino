# typed: strict
# frozen_string_literal: true

class BacklinkListRepository < ApplicationRepository
  sig do
    params(
      user_record: T.nilable(UserRecord),
      page_record: PageRecord,
      before: T.nilable(String),
      after: T.nilable(String),
      limit: Integer
    ).returns(BacklinkList)
  end
  def to_model(user_record:, page_record:, before: nil, after: nil, limit: 15)
    cursor_paginate_page = page_record.backlinked_page_records(user_record:)
      .cursor_paginate(
        after:,
        before:,
        limit:,
        order: {modified_at: :desc, id: :desc}
      ).fetch

    backlinks = cursor_paginate_page.records.map do |page_record|
      Backlink.new(
        page: PageRepository.new.to_model(page_record:)
      )
    end

    BacklinkList.new(
      backlinks:,
      pagination: PaginationRepository.new.to_model(cursor_paginate_page:)
    )
  end
end
