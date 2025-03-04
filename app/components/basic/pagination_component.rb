# typed: strict
# frozen_string_literal: true

module Basic
  class PaginationComponent < ApplicationComponent
    sig { params(pagination_entity: PaginationEntity, previous_path: String, next_path: String).void }
    def initialize(pagination_entity:, previous_path:, next_path:)
      @pagination_entity = T.let(pagination_entity, PaginationEntity)
      @previous_path = T.let(previous_path, String)
      @next_path = T.let(next_path, String)
    end

    sig { returns(PaginationEntity) }
    attr_reader :pagination_entity
    private :pagination_entity

    sig { returns(String) }
    attr_reader :previous_path
    private :previous_path

    sig { returns(String) }
    attr_reader :next_path
    private :next_path
  end
end
