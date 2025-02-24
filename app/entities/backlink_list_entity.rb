# typed: strict
# frozen_string_literal: true

class BacklinkListEntity < ApplicationEntity
  sig { returns(T::Array[BacklinkEntity]) }
  attr_reader :backlink_entities

  sig { returns(PaginationEntity) }
  attr_reader :pagination_entity

  sig do
    params(
      backlink_entities: T::Array[BacklinkEntity],
      pagination_entity: PaginationEntity
    ).void
  end
  def initialize(backlink_entities:, pagination_entity:)
    @backlink_entities = backlink_entities
    @pagination_entity = pagination_entity
  end
end
