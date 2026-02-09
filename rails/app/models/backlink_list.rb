# typed: strict
# frozen_string_literal: true

class BacklinkList < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :backlinks, T::Array[Backlink]
  const :pagination, Pagination
end
