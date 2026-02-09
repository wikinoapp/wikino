# typed: true
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    class DestroyController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(
          user_record: current_user_record!
        )

        unless user_two_factor_auth&.enabled
          flash[:alert] = t("messages.two_factor_auth.not_enabled")
          return redirect_to(settings_two_factor_auth_path)
        end

        form = ::TwoFactorAuths::DestructionForm.new(
          form_params.merge(user_record: current_user_record!)
        )

        if form.invalid?
          return render_component(
            Settings::TwoFactorAuths::ShowView.new(
              current_user: current_user!,
              user_two_factor_auth:,
              form:
            ),
            status: :unprocessable_entity
          )
        end

        ::TwoFactorAuths::DisableService.new.call(user_record: current_user_record!)

        flash[:notice] = t("messages.two_factor_auth.disabled_successfully")
        redirect_to settings_two_factor_auth_path
      end

      sig { returns(ActionController::Parameters) }
      private def form_params
        params.require(:two_factor_auths_destruction_form).permit(
          :password
        )
      end
    end
  end
end
