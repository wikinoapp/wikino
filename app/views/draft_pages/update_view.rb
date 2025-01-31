# typed: strict
# frozen_string_literal: true

module DraftPages
  class UpdateView < ApplicationView
    sig { params(draft_page: DraftPage, link_collection: LinkCollection, backlink_collection: BacklinkCollection).void }
    def initialize(draft_page:, link_collection:, backlink_collection:)
      @draft_page = draft_page
      @link_collection = link_collection
      @backlink_collection = backlink_collection
    end

    sig { returns(DraftPage) }
    attr_reader :draft_page
    private :draft_page

    sig { returns(LinkCollection) }
    attr_reader :link_collection
    private :link_collection

    sig { returns(BacklinkCollection) }
    attr_reader :backlink_collection
    private :backlink_collection
  end
end
