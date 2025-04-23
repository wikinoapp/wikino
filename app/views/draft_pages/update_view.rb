# typed: strict
# frozen_string_literal: true

module DraftPages
  class UpdateView < ApplicationView
    sig do
      params(
        draft_page: DraftPage,
        link_list_entity: LinkListEntity,
        backlink_list_entity: BacklinkListEntity
      ).void
    end
    def initialize(draft_page_entity:, link_list_entity:, backlink_list_entity:)
      @draft_page_entity = draft_page_entity
      @link_list_entity = link_list_entity
      @backlink_list_entity = backlink_list_entity
    end

    sig { returns(DraftPageEntity) }
    attr_reader :draft_page_entity
    private :draft_page_entity

    sig { returns(LinkListEntity) }
    attr_reader :link_list_entity
    private :link_list_entity

    sig { returns(BacklinkListEntity) }
    attr_reader :backlink_list_entity
    private :backlink_list_entity
  end
end
