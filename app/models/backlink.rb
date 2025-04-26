# typed: strict
# frozen_string_literal: true

class Backlink < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :page, Page
end
