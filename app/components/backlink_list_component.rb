# typed: strict
# frozen_string_literal: true

class BacklinkListComponent < ApplicationComponent
  sig { params(page_entity: PageEntity, backlink_list_entity: BacklinkListEntity).void }
  def initialize(page_entity:, backlink_list_entity:)
    @page_entity = T.let(page_entity, PageEntity)
    @backlink_list_entity = T.let(backlink_list_entity, BacklinkListEntity)
  end

  sig { returns(PageEntity) }
  attr_reader :page_entity
  private :page_entity

  sig { returns(BacklinkListEntity) }
  attr_reader :backlink_list_entity
  private :backlink_list_entity

  delegate :space_entity, to: :page_entity
  delegate :backlink_entities, :pagination_entity, to: :backlink_list_entity
end
