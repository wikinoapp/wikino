# typed: strict
# frozen_string_literal: true

class BacklinkList < T::Struct
  const :backlinks, T::Array[Backlink]
  const :page_info, PageInfo
end
