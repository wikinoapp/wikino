# typed: strict
# frozen_string_literal: true

class LinkList < T::Struct
  const :page, Page
  const :links, T::Array[Link]
  const :pagination, Pagination
end
