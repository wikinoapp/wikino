# typed: strict
# frozen_string_literal: true

class BacklinkListEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :backlink_entities, T::Array[BacklinkEntity]
  const :pagination_entity, PaginationEntity
end
