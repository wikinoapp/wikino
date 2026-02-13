# typed: strict
# frozen_string_literal: true

module Dropdowns
  class TrashActionsComponent < ApplicationComponent
    sig { params(page: PageRecord).void }
    def initialize(page:)
      @page = page
    end

    sig { returns(PageRecord) }
    attr_reader :page
    private :page
  end
end
