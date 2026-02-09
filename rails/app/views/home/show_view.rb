# typed: strict
# frozen_string_literal: true

module Home
  class ShowView < ApplicationView
    sig { params(active_spaces: T::Array[Space], current_user: User).void }
    def initialize(active_spaces:, current_user:)
      @active_spaces = active_spaces
      @current_user = current_user
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.home.show")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(T::Array[Space]) }
    attr_reader :active_spaces
    private :active_spaces

    sig { returns(User) }
    attr_reader :current_user
    private :current_user

    sig { returns(PageName) }
    private def current_page_name
      PageName::Home
    end
  end
end
