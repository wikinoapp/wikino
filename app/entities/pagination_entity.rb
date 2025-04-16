# typed: strict
# frozen_string_literal: true

class PaginationEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :next_cursor, T.nilable(String)
  const :has_next, T::Boolean
  const :previous_cursor, T.nilable(String)
  const :has_previous, T::Boolean

  alias_method :has_next?, :has_next
  alias_method :has_previous?, :has_previous

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
