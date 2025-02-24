# typed: strict
# frozen_string_literal: true

class PageConnection < T::Struct
  const :page_entities, T::Array[PageEntity]
  const :pagination, Pagination
end
