# typed: strict
# frozen_string_literal: true

module Dropdowns
  class SpaceOptionsComponent < ApplicationComponent
    sig { params(signed_in: T::Boolean, space: Space).void }
    def initialize(signed_in:, space:)
      @signed_in = signed_in
      @space = space
    end

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in
    alias_method :signed_in?, :signed_in

    sig { returns(Space) }
    attr_reader :space
    private :space
  end
end
