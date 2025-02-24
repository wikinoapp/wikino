# typed: strict
# frozen_string_literal: true

module Pages
  class ShowView < ApplicationView
    sig { params(signed_in: T::Boolean, page_entity: PageEntity, link_collection: LinkCollection, backlink_collection: BacklinkCollection).void }
    def initialize(signed_in:, page_entity:, link_collection:, backlink_collection:)
      @signed_in = signed_in
      @page_entity = page_entity
      @link_collection = link_collection
      @backlink_collection = backlink_collection
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.pages.show", space_name: space_entity.name, page_title: page_entity.title)
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in
    alias_method :signed_in?, :signed_in

    sig { returns(PageEntity) }
    attr_reader :page_entity
    private :page_entity

    sig { returns(LinkCollection) }
    attr_reader :link_collection
    private :link_collection

    sig { returns(BacklinkCollection) }
    attr_reader :backlink_collection
    private :backlink_collection

    delegate :space_entity, :topic_entity, to: :page_entity

    sig { returns(PageName) }
    private def current_page_name
      PageName::PageDetail
    end
  end
end
