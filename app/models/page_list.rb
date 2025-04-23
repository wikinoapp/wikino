# typed: strict
# frozen_string_literal: true

class PageList < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :pages, T::Array[Page]
  const :pagination, Pagination
end
