# typed: strict
# frozen_string_literal: true

module Spaces
  class NewView < ApplicationView
    sig do
      params(
        current_user: User,
        form: Spaces::CreationForm
      ).void
    end
    def initialize(current_user:, form:)
      @current_user = current_user
      @form = form
    end

    sig { override.void }
    def before_render
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(User) }
    attr_reader :current_user
    private :current_user

    sig { returns(Spaces::CreationForm) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::SpaceNew
    end

    sig { returns(String) }
    private def title
      I18n.t("meta.title.spaces.new")
    end
  end
end
