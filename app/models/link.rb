# typed: strict
# frozen_string_literal: true

class Link < T::Struct
  include T::Struct::ActsAsComparable

  const :page_entity, PageEntity
  const :backlink_collection, BacklinkCollection
end
