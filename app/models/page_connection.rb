# typed: strict
# frozen_string_literal: true

class PageConnection < T::Struct
  const :pages, T::Array[Page]
  const :pagination, Pagination
end
