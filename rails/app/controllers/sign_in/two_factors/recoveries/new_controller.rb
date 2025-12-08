# typed: true
# frozen_string_literal: true

module SignIn
  module TwoFactors
    module Recoveries
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

          form = UserSessions::TwoFactorRecoveryForm.new

          render_component SignIn::TwoFactors::Recoveries::NewView.new(
            form:
          )
        end
      end
    end
  end
end
