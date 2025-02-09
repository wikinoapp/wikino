# typed: strict
# frozen_string_literal: true

module Basic
  class PaginationComponent < ApplicationComponent
    sig { params(pagination: ::Pagination, previous_path: String, next_path: String).void }
    def initialize(pagination:, previous_path:, next_path:)
      @pagination = T.let(pagination, ::Pagination)
      @previous_path = T.let(previous_path, String)
      @next_path = T.let(next_path, String)
    end

    sig { returns(::Pagination) }
    attr_reader :pagination
    private :pagination

    sig { returns(String) }
    attr_reader :previous_path
    private :previous_path

    sig { returns(String) }
    attr_reader :next_path
    private :next_path

    sig { returns(T::Boolean) }
    def render?
      pagination.has_previous || pagination.has_next
    end
  end
end
