# typed: strict
# frozen_string_literal: true

class LinkEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :page_entity, PageEntity
  const :backlink_list_entity, BacklinkListEntity
end
