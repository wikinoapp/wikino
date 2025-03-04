# typed: strict
# frozen_string_literal: true

class LinkListEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :link_entities, T::Array[LinkEntity]
  const :pagination_entity, PaginationEntity
end
