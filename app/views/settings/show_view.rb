# typed: strict
# frozen_string_literal: true

module Settings
  class ShowView < ApplicationView
    sig do
      params(
        current_user: User
      ).void
    end
    def initialize(current_user:)
      @current_user = current_user
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.settings.show")
      helpers.set_meta_tags(title:, **default_meta_tags(site: false))
    end

    sig { returns(User) }
    attr_reader :current_user
    private :current_user

    sig { returns(PageName) }
    private def current_page_name
      PageName::Settings
    end
  end
end
