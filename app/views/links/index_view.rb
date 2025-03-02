# typed: strict
# frozen_string_literal: true

module Links
  class IndexView < ApplicationView
    sig { params(link_list_entity: LinkListEntity).void }
    def initialize(link_list_entity:)
      @link_list_entity = link_list_entity
    end

    sig { returns(LinkListEntity) }
    attr_reader :link_list_entity
    private :link_list_entity
  end
end
