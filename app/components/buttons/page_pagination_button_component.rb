# typed: strict
# frozen_string_literal: true

module Buttons
  class PagePaginationButtonComponent < ApplicationComponent
    sig { params(page: Page, pagination: Pagination).void }
    def initialize(page:, pagination:)
      @page = page
      @pagination = pagination
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(Pagination) }
    attr_reader :pagination
    private :pagination
  end
end
