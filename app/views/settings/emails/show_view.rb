# typed: strict
# frozen_string_literal: true

module Settings
  module Emails
    class ShowView < ApplicationView
      sig { params(current_user: User, form: Emails::EditForm).void }
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

      sig { returns(Emails::EditForm) }
      attr_reader :form
      private :form

      sig { returns(String) }
      private def title
        I18n.t("meta.title.settings.emails.show")
      end

      sig { returns(PageName) }
      private def current_page_name
        PageName::SettingsEmail
      end
    end
  end
end
