# typed: strict
# frozen_string_literal: true

module Breadcrumbs
  class SpaceBreadcrumbsComponent < ApplicationComponent
    renders_many :items, Breadcrumbs::SpaceBreadcrumbs::ItemComponent

    sig { params(space_entity: SpaceEntity).void }
    def initialize(space_entity:)
      @space_entity = space_entity
    end

    sig { returns(SpaceEntity) }
    attr_reader :space_entity
    private :space_entity
  end
end
