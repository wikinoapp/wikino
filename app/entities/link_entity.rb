# typed: strict
# frozen_string_literal: true

class LinkEntity < ApplicationEntity
  sig { returns(PageEntity) }
  attr_reader :page_entity

  sig { returns(BacklinkListEntity) }
  attr_reader :backlink_list_entity

  sig do
    params(
      page_entity: PageEntity,
      backlink_list_entity: BacklinkListEntity
    ).void
  end
  def initialize(page_entity:, backlink_list_entity:)
    @page_entity = page_entity
    @backlink_list_entity = backlink_list_entity
  end
end
