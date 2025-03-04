# typed: strict
# frozen_string_literal: true

module Links
  class IndexView < ApplicationView
    sig do
      params(
        page_entity: PageEntity,
        link_list_entity: LinkListEntity
      ).void
    end
    def initialize(page_entity:, link_list_entity:)
      @page_entity = page_entity
      @link_list_entity = link_list_entity
    end

    sig { returns(PageEntity) }
    attr_reader :page_entity
    private :page_entity

    sig { returns(LinkListEntity) }
    attr_reader :link_list_entity
    private :link_list_entity
  end
end
