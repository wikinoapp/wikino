# typed: strict
# frozen_string_literal: true

module Dropdowns
  class PageActionsComponent < ApplicationComponent
    sig { params(page_entity: PageEntity).void }
    def initialize(page_entity:)
      @page_entity = page_entity
    end

    sig { returns(PageEntity) }
    attr_reader :page_entity
    private :page_entity

    delegate :space_entity, to: :page_entity
  end
end
