# typed: strict
# frozen_string_literal: true

class Link < T::Struct
  const :page, Page
  const :backlinked_pages, T::Array[Page]
  const :pagination, Pagination
end
