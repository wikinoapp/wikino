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
      form = Accounts::CreationForm.new(
        form_params.merge(
          email: @email_confirmation.not_nil!.email.not_nil!,
          locale: current_locale.serialize,
          time_zone: "Asia/Tokyo" # TODO: あとでユーザーのタイムゾーンを指定する
        )
      )

      if form.invalid?
        return render_component(Accounts::NewView.new(form:), status: :unprocessable_entity)
      end

      account_result = Accounts::CreateService.new.call(
        email: form.email.not_nil!,
        atname: form.atname.not_nil!,
        locale: Locale.deserialize(form.locale),
        password: form.password.not_nil!,
        time_zone: form.time_zone.not_nil!
      )

      user_session_result = UserSessions::CreateService.new.call(
        user_record: account_result.user,
        ip_address: original_remote_ip,
        user_agent: request.user_agent
      )

      sign_in(user_session_result.user_session_record)

      flash[:notice] = t("messages.accounts.signed_up_successfully_html")
      redirect_to after_authentication_url
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:accounts_creation_form), ActionController::Parameters)
        .permit(:atname, :password)
    end
  end
end
