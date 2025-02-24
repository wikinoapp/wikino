# typed: strict
# frozen_string_literal: true

class PageListEntity < ApplicationEntity
  sig { returns(T::Array[PageEntity]) }
  attr_reader :page_entities

  sig { returns(PaginationEntity) }
  attr_reader :pagination_entity

  sig do
    params(
      page_entities: T::Array[PageEntity],
      pagination_entity: PaginationEntity
    ).void
  end
  def initialize(page_entities:, pagination_entity:)
    @page_entities = page_entities
    @pagination_entity = pagination_entity
  end
end
