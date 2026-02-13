# typed: strict
# frozen_string_literal: true

module BaseUI
  class PaginationComponent < ApplicationComponent
    sig { params(pagination: Pagination, previous_path: String, next_path: String).void }
    def initialize(pagination:, previous_path:, next_path:)
      @pagination = pagination
      @previous_path = previous_path
      @next_path = next_path
    end

    sig { returns(Pagination) }
    attr_reader :pagination
    private :pagination

    sig { returns(String) }
    attr_reader :previous_path
    private :previous_path

    sig { returns(String) }
    attr_reader :next_path
    private :next_path
  end
end
