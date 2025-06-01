# typed: true
# frozen_string_literal: true

module UserSessions
  module TwoFactorAuths
    class CreateController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_pending_two_factor_auth

      sig { returns(T.untyped) }
      def call
        form = UserSessionForm::TwoFactorVerification.new(form_params)

        # ペンディング中のユーザーを取得
        user_record = UserRecord.find_by(id: session[:pending_user_id])
        unless user_record
          session.delete(:pending_user_id)
          redirect_to sign_in_path
          return
        end

        # フォームにユーザーレコードを設定
        form.user_record = user_record

        if form.invalid?
          return render_component(
            UserSessions::TwoFactorAuths::NewView.new(form:),
            status: :unprocessable_entity
          )
        end

        # セッション作成
        result = UserSessionService::Create.new.call(
          user_record: user_record,
          ip_address: original_remote_ip,
          user_agent: request.user_agent
        )

        # ペンディングセッションをクリア
        session.delete(:pending_user_id)

        # ログイン完了
        sign_in(result.user_session_record)

        flash[:notice] = t("messages.accounts.signed_in_successfully")
        redirect_to after_authentication_url
      end

      private

      sig { returns(ActionController::Parameters) }
      def form_params
        params.require(:user_session_form_two_factor_verification).permit(
          :code
        )
      end

      sig { void }
      def require_pending_two_factor_auth
        if session[:pending_user_id].blank?
          redirect_to sign_in_path
        end
      end
    end
  end
end
