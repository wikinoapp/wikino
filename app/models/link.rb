# typed: strict
# frozen_string_literal: true

class Link < T::Struct
  include T::Struct::ActsAsComparable

  const :page, Page
  const :backlink_collection, BacklinkCollection
end
