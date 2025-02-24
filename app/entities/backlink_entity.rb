# typed: strict
# frozen_string_literal: true

class BacklinkEntity < ApplicationEntity
  sig { returns(PageEntity) }
  attr_reader :page_entity

  sig { params(page_entity: PageEntity).void }
  def initialize(page_entity:)
    @page_entity = page_entity
  end
end
