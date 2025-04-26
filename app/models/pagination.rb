# typed: strict
# frozen_string_literal: true

class Pagination < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :next_cursor, T.nilable(String)
  const :has_next, T::Boolean
  const :previous_cursor, T.nilable(String)
  const :has_previous, T::Boolean

  alias_method :has_next?, :has_next
  alias_method :has_previous?, :has_previous
end
