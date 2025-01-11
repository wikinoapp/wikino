# typed: strict
# frozen_string_literal: true

module Buttons
  class LinkPaginationComponent < ApplicationComponent
    sig { params(path: String).void }
    def initialize(path:)
      @path = path
    end

    sig { returns(String) }
    attr_reader :path
    private :path
  end
end
