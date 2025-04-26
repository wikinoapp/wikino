# typed: strict
# frozen_string_literal: true

class PageListRepository < ApplicationRepository
  sig do
    params(
      space_record: SpaceRecord,
      before: T.nilable(String),
      after: T.nilable(String)
    ).returns(PageList)
  end
  def restorable(space_record:, before:, after:)
    cursor_paginate_page = space_record.page_records.preload(:topic_record).restorable.cursor_paginate(
      before: before.presence,
      after: after.presence,
      limit: 100,
      order: {trashed_at: :desc, id: :desc}
    ).fetch

    pages = PageRepository.new.to_models(page_records: cursor_paginate_page.records)
    pagination = PaginationRepository.new.to_model(cursor_paginate_page:)

    PageList.new(pages:, pagination:)
  end
end
