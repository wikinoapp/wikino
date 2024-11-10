# typed: true
# frozen_string_literal: true

module Accounts
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::EmailConfirmationFindable

    around_action :set_locale
    before_action :require_no_authentication
    before_action :require_succeeded_email_confirmation

    sig { returns(T.untyped) }
    def call
      @form = AccountForm.new(
        form_params.merge(
          email: @email_confirmation.not_nil!.email.not_nil!,
          locale: current_locale.serialize,
          time_zone: "Asia/Tokyo" # TODO: あとでユーザーのタイムゾーンを指定する
        )
      )

      if @form.invalid?
        return render("accounts/new/call", status: :unprocessable_entity)
      end

      account_result = CreateAccountUseCase.new.call(
        space_identifier: @form.space_identifier.not_nil!,
        email: @form.email.not_nil!,
        atname: @form.atname.not_nil!,
        locale: UserLocale.deserialize(@form.locale),
        password: @form.password.not_nil!,
        time_zone: @form.time_zone.not_nil!
      )

      session_result = CreateSessionUseCase.new.call(
        user: account_result.user,
        ip_address: original_remote_ip,
        user_agent: request.user_agent
      )

      sign_in(session_result.session)

      flash[:notice] = t("messages.accounts.signed_up_successfully")
      redirect_to after_authentication_url
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:account_form), ActionController::Parameters).permit(:password)
    end
  end
end
