# typed: strict
# frozen_string_literal: true

module Navbars
  class BottomComponent < ApplicationComponent
    sig { params(current_page_name: PageName).void }
    def initialize(current_page_name:)
      @current_page_name = current_page_name
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name
  end
end
