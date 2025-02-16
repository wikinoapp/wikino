# typed: strict
# frozen_string_literal: true

module Home
  class ShowView < ApplicationView
    sig { params(active_spaces: Space::PrivateCollectionProxy).void }
    def initialize(active_spaces:)
      @active_spaces = active_spaces
    end

    def before_render
      title = I18n.t("meta.title.home.show")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(Space::PrivateCollectionProxy) }
    attr_reader :active_spaces
    private :active_spaces

    sig { returns(PageName) }
    private def current_page_name
      PageName::Home
    end
  end
end
