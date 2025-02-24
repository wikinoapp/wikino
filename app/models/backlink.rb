# typed: strict
# frozen_string_literal: true

class Backlink < T::Struct
  include T::Struct::ActsAsComparable

  const :page_entity, PageEntity
end
