# typed: strict
# frozen_string_literal: true

class Link < T::Struct
  const :note, Note
  const :backlinked_notes, T::Array[Note]
  const :page_info, PageInfo
end
