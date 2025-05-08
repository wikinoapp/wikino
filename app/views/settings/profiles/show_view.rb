# typed: strict
# frozen_string_literal: true

module Settings
  module Profiles
    class ShowView < ApplicationView
      sig do
        params(
          current_user: User,
          form: ProfileForm::Edit
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

      sig { returns(ProfileForm::Edit) }
      attr_reader :form
      private :form

      sig { returns(String) }
      private def title
        I18n.t("meta.title.settings.profiles.show")
      end

      sig { returns(PageName) }
      private def current_page_name
        PageName::SettingsProfile
      end
    end
  end
end
