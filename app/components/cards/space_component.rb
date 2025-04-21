# typed: strict
# frozen_string_literal: true

module Cards
  class SpaceComponent < ApplicationComponent
    sig { params(space_entity: SpaceEntity, class_name: String).void }
    def initialize(space_entity:, class_name: "")
      @space_entity = space_entity
      @class_name = class_name
    end

    sig { returns(SpaceEntity) }
    attr_reader :space_entity
    private :space_entity

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
