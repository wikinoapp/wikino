# typed: strict
# frozen_string_literal: true

module Welcome
  class ShowView < ApplicationView
    sig { override.void }
    def before_render
      title = I18n.t("meta.title.welcome.show")
      helpers.set_meta_tags(title:, **default_meta_tags, reverse: false)
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::Welcome
    end
  end
end
