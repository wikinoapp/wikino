# typed: true
# frozen_string_literal: true

module UserSessions
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_no_authentication

    sig { returns(T.untyped) }
    def call
      form = UserSessionForm.new(form_params)

      if form.invalid?
        return render(SignIn::ShowView.new(form:), status: :unprocessable_entity)
      end

      result = CreateUserSessionService.new.call(
        user: form.user.not_nil!,
        ip_address: original_remote_ip,
        user_agent: request.user_agent
      )

      sign_in(result.user_session)

      flash[:notice] = t("messages.accounts.signed_in_successfully")
      redirect_to after_authentication_url
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:user_session_form), ActionController::Parameters).permit(
        :space_identifier,
        :email,
        :password
      )
    end
  end
end
