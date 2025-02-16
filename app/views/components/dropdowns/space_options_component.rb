# typed: strict
# frozen_string_literal: true

module Dropdowns
  class SpaceOptionsComponent < ApplicationComponent
    sig { params(space: Space).void }
    def initialize(space:)
      @space = space
    end

    sig { returns(Space) }
    attr_reader :space
    private :space
  end
end
