# typed: strict
# frozen_string_literal: true

module Welcome
  class ShowView < ApplicationView
    use_helpers :set_meta_tags

    sig { void }
    def initialize
      @current_page_name = PageName::Welcome
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name
  end
end
