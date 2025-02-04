# typed: strict
# frozen_string_literal: true

module Home
  class ShowView < ApplicationView
    sig { params(active_spaces: Space::PrivateCollectionProxy).void }
    def initialize(active_spaces:)
      @active_spaces = active_spaces
    end

    sig { returns(Space::PrivateCollectionProxy) }
    attr_reader :active_spaces
    private :active_spaces
  end
end
