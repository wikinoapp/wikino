# typed: true
# frozen_string_literal: true

module AccountSwitches
  class CreateController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      params[:user_id].in?(cookie_user_ids) || raise(ActionController::RoutingError, "Not Found")

      user = User.kept.find(params[:user_id])
      result = CreateUserSessionUseCase.new.call(
        user:,
        ip_address: original_remote_ip,
        user_agent: request.user_agent
      )

      sign_in(result.session)

      flash[:notice] = t("messages.accounts.account_switched_successfully")
      redirect_to space_path(Current.user!.space.not_nil!.identifier)
    end
  end
end
