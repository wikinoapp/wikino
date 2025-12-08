# typed: strict
# frozen_string_literal: true

class LinkList < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :links, T::Array[Link]
  const :pagination, Pagination
end
