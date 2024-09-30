# typed: strict
# frozen_string_literal: true

module Cards
  class PageCardComponent < ApplicationComponent
    sig { params(page: Page, class_name: String).void }
    def initialize(page:, class_name: "")
      @page = page
      @class_name = class_name
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
