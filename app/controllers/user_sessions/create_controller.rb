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
      form = UserSessionForm::Creation.new(form_params)

      if form.invalid?
        return render_component(SignIn::ShowView.new(form:), status: :unprocessable_entity)
      end

      result = UserSessionService::Create.new.call(
        user_record: form.user_record.not_nil!,
        ip_address: original_remote_ip,
        user_agent: request.user_agent
      )

      sign_in(result.user_session_record)

      flash[:notice] = t("messages.accounts.signed_in_successfully")
      redirect_to after_authentication_url
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:user_session_form_creation), ActionController::Parameters).permit(
        :email,
        :password
      )
    end
  end
end
