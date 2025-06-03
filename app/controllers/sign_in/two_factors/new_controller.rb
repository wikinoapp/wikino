# typed: true
# frozen_string_literal: true

module SignIn
  module TwoFactors
    class NewController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_no_authentication

      sig { returns(T.untyped) }
      def call
        if session[:pending_user_id].blank?
          return redirect_to(sign_in_path)
        end

        form = UserSessionForm::TwoFactorVerification.new

        render_component SignIn::TwoFactors::NewView.new(
          form:
        )
      end
    end
  end
end
