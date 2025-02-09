# typed: strict
# frozen_string_literal: true

module Pages
  class ShowView < ApplicationView
    use_helpers :set_meta_tags

    sig { params(page: Page, link_collection: LinkCollection, backlink_collection: BacklinkCollection).void }
    def initialize(page:, link_collection:, backlink_collection:)
      @current_page_name = PageName::PageDetail
      @page = page
      @link_collection = link_collection
      @backlink_collection = backlink_collection
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(LinkCollection) }
    attr_reader :link_collection
    private :link_collection

    sig { returns(BacklinkCollection) }
    attr_reader :backlink_collection
    private :backlink_collection

    sig { returns(Space) }
    def space
      page.space.not_nil!
    end

    sig { returns(Topic) }
    def topic
      page.topic.not_nil!
    end
  end
end
