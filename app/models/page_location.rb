# typed: strict
# frozen_string_literal: true

class PageLocation < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :key, PageLocationKey
  const :topic, Topic
  const :page, Page
end
