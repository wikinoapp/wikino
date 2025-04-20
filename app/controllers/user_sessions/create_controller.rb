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
      user_session = UserSession.new(form_params.merge(
        ip_address: original_remote_ip,
        user_agent: request.user_agent
      ))

      user_session = UserSessionRepository.new.create(user_session:)

      if user_session.invalid?
        return render(SignIn::ShowView.new(user_session:), status: :unprocessable_entity)
      end

      sign_in(user_session)

      flash[:notice] = t("messages.accounts.signed_in_successfully")
      redirect_to after_authentication_url
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:user_session_form), ActionController::Parameters).permit(
        :email,
        :password
      )
    end
  end
end
