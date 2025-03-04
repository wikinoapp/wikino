# typed: strict
# frozen_string_literal: true

class PageListEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :page_entities, T::Array[PageEntity]
  const :pagination_entity, PaginationEntity
end
