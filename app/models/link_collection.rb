# typed: strict
# frozen_string_literal: true

class LinkCollection < T::Struct
  const :page_entity, PageEntity
  const :links, T::Array[Link]
  const :pagination, Pagination
end
