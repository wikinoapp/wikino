# typed: strict
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    class NewView < ApplicationView
      sig do
        params(
          current_user: User,
          secret: String,
          provisioning_uri: String,
          qr_code: T.nilable(String),
          form: ::TwoFactorAuths::CreationForm
        ).void
      end
      def initialize(current_user:, secret:, provisioning_uri:, qr_code:, form:)
        @current_user = current_user
        @secret = secret
        @provisioning_uri = provisioning_uri
        @qr_code = qr_code
        @form = form
      end

      sig { override.void }
      def before_render
        helpers.set_meta_tags(title:, **default_meta_tags)
      end

      sig { returns(User) }
      attr_reader :current_user
      private :current_user

      sig { returns(String) }
      attr_reader :secret
      private :secret

      sig { returns(String) }
      attr_reader :provisioning_uri
      private :provisioning_uri

      sig { returns(T.nilable(String)) }
      attr_reader :qr_code
      private :qr_code

      sig { returns(::TwoFactorAuths::CreationForm) }
      attr_reader :form
      private :form

      sig { returns(String) }
      private def title
        t("meta.title.settings.two_factor_auth.new")
      end

      sig { returns(PageName) }
      private def current_page_name
        PageName::SettingsTwoFactorAuthNew
      end
    end
  end
end
