# typed: strict
# frozen_string_literal: true

class BacklinkCollection < T::Struct
  const :page_entity, PageEntity
  const :backlinks, T::Array[Backlink]
  const :pagination, Pagination
end
