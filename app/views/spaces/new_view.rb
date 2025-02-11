# typed: strict
# frozen_string_literal: true

module Spaces
  class NewView < ApplicationView
    use_helpers :set_meta_tags

    sig { params(form: NewSpaceForm).void }
    def initialize(form:)
      @form = form
    end

    sig { returns(NewSpaceForm) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::SpaceNew
    end
  end
end
