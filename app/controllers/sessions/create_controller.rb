# typed: true
# frozen_string_literal: true

module Sessions
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_no_authentication

    sig { returns(T.untyped) }
    def call
      @form = SessionForm.new(form_params)

      if @form.invalid?
        return render("sign_in/show/call", status: :unprocessable_entity)
      end

      result = CreateSessionUseCase.new.call(
        user: @form.user.not_nil!,
        ip_address: original_remote_ip,
        user_agent: request.user_agent
      )

      sign_in(result.session)

      flash[:notice] = t("messages.accounts.signed_in_successfully")
      redirect_to after_authentication_url
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:session_form), ActionController::Parameters).permit(
        :space_identifier,
        :email,
        :password
      )
    end
  end
end
