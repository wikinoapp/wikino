# typed: strict
# frozen_string_literal: true

module Footers
  class PageComponent < ApplicationComponent
    sig { params(page_entity: PageEntity, link_list_entity: LinkListEntity, backlink_list_entity: BacklinkListEntity).void }
    def initialize(page_entity:, link_list_entity:, backlink_list_entity:)
      @page_entity = page_entity
      @link_list_entity = link_list_entity
      @backlink_list_entity = backlink_list_entity
    end

    sig { returns(PageEntity) }
    attr_reader :page_entity
    private :page_entity

    sig { returns(LinkListEntity) }
    attr_reader :link_list_entity
    private :link_list_entity

    sig { returns(BacklinkListEntity) }
    attr_reader :backlink_list_entity
    private :backlink_list_entity

    sig { returns(T::Boolean) }
    def render?
      show_links? || show_backlinks?
    end

    sig { returns(T::Boolean) }
    private def show_links?
      link_list_entity.link_entities.present?
    end

    sig { returns(T::Boolean) }
    private def show_backlinks?
      backlink_list_entity.backlink_entities.present?
    end
  end
end
