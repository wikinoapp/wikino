# typed: strict
# frozen_string_literal: true

module Links
  class IndexView < ApplicationView
    sig { params(link_collection: LinkCollection).void }
    def initialize(link_collection:)
      @link_collection = link_collection
    end

    sig { returns(LinkCollection) }
    attr_reader :link_collection
    private :link_collection
  end
end
