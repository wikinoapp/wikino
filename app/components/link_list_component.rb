# typed: strict
# frozen_string_literal: true

class LinkListComponent < ApplicationComponent
  sig { params(page_entity: PageEntity, link_list_entity: LinkListEntity).void }
  def initialize(page_entity:, link_list_entity:)
    @page_entity = T.let(page_entity, PageEntity)
    @link_list_entity = T.let(link_list_entity, LinkListEntity)
  end

  sig { returns(PageEntity) }
  attr_reader :page_entity
  private :page_entity

  sig { returns(LinkListEntity) }
  attr_reader :link_list_entity
  private :link_list_entity

  delegate :space_entity, to: :page_entity
  delegate :link_entities, :pagination_entity, to: :link_list_entity
end
