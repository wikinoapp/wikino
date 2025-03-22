# typed: strict
# frozen_string_literal: true

module Settings
  module Profiles
    class UpdateController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        current_user = T.let(Current.viewer!, User)

        form = EditProfileForm.new(form_params.merge(user: current_user))

        if form.invalid?
          return render(
            Settings::Profiles::ShowView.new(
              current_user_entity: current_user.to_entity,
              form:
            ),
            status: :unprocessable_entity
          )
        end

        UpdateProfileService.new.call(form:)

        flash[:notice] = t("messages.profiles.updated")
        redirect_to settings_profile_path
      end

      sig { returns(ActionController::Parameters) }
      private def form_params
        params.require(:edit_profile_form).permit(
          :atname,
          :name,
          :description
        )
      end
    end
  end
end
