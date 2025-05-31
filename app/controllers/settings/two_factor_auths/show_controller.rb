# typed: strict
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    class ShowController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(user_record: current_user_record!)

        render_component Settings::TwoFactorAuths::ShowView.new(
          current_user: current_user!,
          user_two_factor_auth:
        )
      end
    end
  end
end

