# typed: strict
# frozen_string_literal: true

module Footers
  class PageFooterComponent < ApplicationComponent
    sig { params(link_collection: LinkCollection, backlink_collection: BacklinkCollection).void }
    def initialize(link_collection:, backlink_collection:)
      @link_collection = link_collection
      @backlink_collection = backlink_collection
    end

    sig { returns(LinkCollection) }
    attr_reader :link_collection
    private :link_collection

    sig { returns(BacklinkCollection) }
    attr_reader :backlink_collection
    private :backlink_collection
  end
end
