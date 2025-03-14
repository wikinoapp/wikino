# typed: strict
# frozen_string_literal: true

module Dropdowns
  class SpaceOptionsComponent < ApplicationComponent
    sig { params(signed_in: T::Boolean, space_entity: SpaceEntity).void }
    def initialize(signed_in:, space_entity:)
      @signed_in = signed_in
      @space_entity = space_entity
    end

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in
    alias_method :signed_in?, :signed_in

    sig { returns(SpaceEntity) }
    attr_reader :space_entity
    private :space_entity
  end
end
