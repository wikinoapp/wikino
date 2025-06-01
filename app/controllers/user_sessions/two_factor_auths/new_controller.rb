# typed: true
# frozen_string_literal: true

module UserSessions
  module TwoFactorAuths
    class NewController < ApplicationController
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_pending_two_factor_auth

      sig { returns(T.untyped) }
      def call
        render_component UserSessions::TwoFactorAuths::NewView.new(
          form: UserSessionForm::TwoFactorVerification.new
        )
      end

      private

      sig { void }
      def require_pending_two_factor_auth
        if session[:pending_user_id].blank?
          redirect_to sign_in_path
        end
      end
    end
  end
end
