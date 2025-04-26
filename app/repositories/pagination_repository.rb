# typed: strict
# frozen_string_literal: true

class PaginationRepository < ApplicationRepository
  sig { params(cursor_paginate_page: ActiveRecordCursorPaginate::Page).returns(Pagination) }
  def to_model(cursor_paginate_page:)
    Pagination.new(
      next_cursor: cursor_paginate_page.has_next? ? cursor_paginate_page.next_cursor : nil,
      has_next: cursor_paginate_page.has_next?,
      has_previous: cursor_paginate_page.has_previous?,
      previous_cursor: cursor_paginate_page.has_previous? ? cursor_paginate_page.previous_cursor : nil
    )
  end
end
