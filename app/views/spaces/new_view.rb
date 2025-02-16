# typed: strict
# frozen_string_literal: true

module Spaces
  class NewView < ApplicationView
    sig { params(form: NewSpaceForm).void }
    def initialize(form:)
      @form = form
    end

    def before_render
      title = I18n.t("meta.title.spaces.new")
      helpers.set_meta_tags(title:, **default_meta_tags)
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
