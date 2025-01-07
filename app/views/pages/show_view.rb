# typed: strict
# frozen_string_literal: true

module Pages
  class ShowView < ApplicationView
    use_helpers :policy

    sig { params(page: Page, link_collection: LinkCollection, backlink_collection: BacklinkCollection).void }
    def initialize(page:, link_collection:, backlink_collection:)
      @page = page
      @link_collection = link_collection
      @backlink_collection = backlink_collection
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(LinkCollection) }
    attr_reader :link_collection
    private :link_collection

    sig { returns(BacklinkCollection) }
    attr_reader :backlink_collection
    private :backlink_collection
  end
end
