# typed: strict
# frozen_string_literal: true

module Dropdowns
  class SpaceOptionsComponent < ApplicationComponent
    sig { params(joined_space: T::Boolean, space: Space).void }
    def initialize(joined_space:, space:)
      @joined_space = joined_space
      @space = space
    end

    sig { returns(T::Boolean) }
    attr_reader :joined_space
    private :joined_space
    alias_method :joined_space?, :joined_space

    sig { returns(Space) }
    attr_reader :space
    private :space
  end
end
