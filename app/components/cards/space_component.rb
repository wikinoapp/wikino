# typed: strict
# frozen_string_literal: true

module Cards
  class SpaceComponent < ApplicationComponent
    sig { params(space: Space, class_name: String).void }
    def initialize(space:, class_name: "")
      @space = space
      @class_name = class_name
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
