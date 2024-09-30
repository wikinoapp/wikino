# typed: strict
# frozen_string_literal: true

class Pagination
  extend T::Sig

  sig { returns(T.nilable(String)) }
  attr_reader :next_cursor

  sig { returns(T::Boolean) }
  attr_reader :has_next

  sig { returns(T::Boolean) }
  attr_reader :has_previous

  sig { returns(T.nilable(String)) }
  attr_reader :previous_cursor

  sig do
    params(
      next_cursor: T.nilable(String),
      has_next: T::Boolean,
      has_previous: T::Boolean,
      previous_cursor: T.nilable(String)
    ).void
  end
  def initialize(next_cursor:, has_next:, has_previous:, previous_cursor:)
    @next_cursor = next_cursor
    @has_next = has_next
    @has_previous = has_previous
    @previous_cursor = previous_cursor
  end

  sig { params(cursor_paginate_page: ActiveRecordCursorPaginate::Page).returns(T.attached_class) }
  def self.from_cursor_paginate(cursor_paginate_page:)
    new(
      next_cursor: cursor_paginate_page.has_next? ? cursor_paginate_page.next_cursor : nil,
      has_next: cursor_paginate_page.has_next?,
      has_previous: cursor_paginate_page.has_previous?,
      previous_cursor: cursor_paginate_page.has_previous? ? cursor_paginate_page.previous_cursor : nil
    )
  end
end
