# typed: true
# frozen_string_literal: true

module Settings
  module Account
    module Deletions
      class CreateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          form = Accounts::DestroyConfirmationForm.new(
            form_params.merge(user_record: current_user_record!)
          )

          if form.invalid?
            return render_component(
              Settings::Account::Deletions::NewView.new(
                current_user: current_user!,
                form:,
                # アクティブなスペースがないときこのアクションが実行できるので
                # 決め打ちで空の配列を渡している
                active_spaces: []
              ),
              status: :unprocessable_entity
            )
          end

          AccountService::SoftDestroy.new.call(user_record: current_user_record!)

          sign_out

          flash[:notice] = t("messages.settings.account.deletions.deleted")
          redirect_to root_path
        end

        sig { returns(ActionController::Parameters) }
        private def form_params
          T.cast(params.require(:accounts_destroy_confirmation_form), ActionController::Parameters).permit(
            :user_atname
          )
        end
      end
    end
  end
end
