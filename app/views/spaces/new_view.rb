# typed: strict
# frozen_string_literal: true

module Spaces
  class NewView < ApplicationView
    sig do
      params(
        current_user_entity: UserEntity,
        form: NewSpaceForm
      ).void
    end
    def initialize(current_user_entity:, form:)
      @current_user_entity = current_user_entity
      @form = form
    end

    sig { override.void }
    def before_render
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(UserEntity) }
    attr_reader :current_user_entity
    private :current_user_entity

    sig { returns(NewSpaceForm) }
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
