# typed: strict
# frozen_string_literal: true

class LinkListEntity < ApplicationEntity
  sig { returns(T::Array[LinkEntity]) }
  attr_reader :link_entities

  sig { returns(PaginationEntity) }
  attr_reader :pagination_entity

  sig do
    params(
      link_entities: T::Array[LinkEntity],
      pagination_entity: PaginationEntity
    ).void
  end
  def initialize(link_entities:, pagination_entity:)
    @link_entities = link_entities
    @pagination_entity = pagination_entity
  end
end
