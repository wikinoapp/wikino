# typed: strict
# frozen_string_literal: true

class PageInfo
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

  sig { params(page: ActiveRecordCursorPaginate::Page).returns(T.attached_class) }
  def self.from_cursor_paginate_page(page:)
    new(
      next_cursor: page.has_next? ? page.next_cursor : nil,
      has_next: page.has_next?,
      has_previous: page.has_previous?,
      previous_cursor: page.has_previous? ? page.previous_cursor : nil
    )
  end
end
