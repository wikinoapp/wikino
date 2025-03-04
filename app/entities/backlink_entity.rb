# typed: strict
# frozen_string_literal: true

class BacklinkEntity < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :page_entity, PageEntity
end
