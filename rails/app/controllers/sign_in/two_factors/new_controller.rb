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
        pending_user_record = UserRecord.visible.find_by(id: session[:pending_user_id])

        unless pending_user_record&.two_factor_enabled?
          return redirect_to(sign_in_path)
        end

        form = UserSessions::TwoFactorVerificationForm.new

        render_component SignIn::TwoFactors::NewView.new(
          form:
        )
      end
    end
  end
end
