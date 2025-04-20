# typed: strict
# frozen_string_literal: true

module SignIn
  class ShowView < ApplicationView
    sig { params(user_session: UserSession).void }
    def initialize(user_session:)
      @user_session = user_session
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.sign_in.show")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(UserSession) }
    attr_reader :user_session
    private :user_session

    sig { returns(PageName) }
    private def current_page_name
      PageName::SignIn
    end
  end
end
