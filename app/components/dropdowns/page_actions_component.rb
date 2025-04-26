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

    delegate :space, to: :page
  end
end
