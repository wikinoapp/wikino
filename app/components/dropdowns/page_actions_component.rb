# typed: strict
# frozen_string_literal: true

module Dropdowns
  class PageActionsComponent < ApplicationComponent
    sig { params(page: Page).void }
    def initialize(page:)
      @page = page
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(Space) }
    def space
      page.space.not_nil!
    end
  end
end
