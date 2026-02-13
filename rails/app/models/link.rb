# typed: strict
# frozen_string_literal: true

class Link < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :page, Page
  const :backlink_list, BacklinkList
end
