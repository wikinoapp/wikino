# typed: strict
# frozen_string_literal: true

module Spaces
  class NewView < ApplicationView
    use_helpers :set_meta_tags

    sig { void }
    def initialize
      @current_page_name = PageName::SpaceNew
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name
  end
end
