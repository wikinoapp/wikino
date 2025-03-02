# typed: strict
# frozen_string_literal: true

module Backlinks
  class IndexView < ApplicationView
    sig { params(page_entity: PageEntity, backlink_list_entity: BacklinkListEntity).void }
    def initialize(page_entity:, backlink_list_entity:)
      @page_entity = page_entity
      @backlink_list_entity = backlink_list_entity
    end

    sig { returns(PageEntity) }
    attr_reader :page_entity
    private :page_entity

    sig { returns(BacklinkListEntity) }
    attr_reader :backlink_list_entity
    private :backlink_list_entity
  end
end
