# typed: strict
# frozen_string_literal: true

module Welcome
  class ShowView < ApplicationView
    use_helpers :set_meta_tags

    sig { returns(PageName) }
    private def current_page_name
      PageName::Welcome
    end
  end
end
