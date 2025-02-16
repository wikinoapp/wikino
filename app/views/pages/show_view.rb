# typed: strict
# frozen_string_literal: true

module Pages
  class ShowView < ApplicationView
    sig { params(page: Page, link_collection: LinkCollection, backlink_collection: BacklinkCollection).void }
    def initialize(page:, link_collection:, backlink_collection:)
      @page = page
      @link_collection = link_collection
      @backlink_collection = backlink_collection
    end

    def before_render
      title = I18n.t("meta.title.pages.show", space_name: space.name, page_title: page.title)
      helpers.set_meta_tags(title:, **default_meta_tags)
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

    sig { returns(Space) }
    def space
      page.space.not_nil!
    end

    sig { returns(Topic) }
    def topic
      page.topic.not_nil!
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::PageDetail
    end
  end
end
