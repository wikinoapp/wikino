# typed: strict
# frozen_string_literal: true

module Spaces
  class NewView < ApplicationView
    use_helpers :set_meta_tags

    sig { returns(PageName) }
    private def current_page_name
      PageName::SpaceNew
    end
  end
end
