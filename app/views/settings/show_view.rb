# typed: strict
# frozen_string_literal: true

module Settings
  class ShowView < ApplicationView
    sig do
      params(
        current_user_entity: UserEntity
      ).void
    end
    def initialize(current_user_entity:)
      @current_user_entity = current_user_entity
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.settings.show")
      helpers.set_meta_tags(title:, **default_meta_tags(site: false))
    end

    sig { returns(UserEntity) }
    attr_reader :current_user_entity
    private :current_user_entity

    sig { returns(PageName) }
    private def current_page_name
      PageName::Settings
    end
  end
end
