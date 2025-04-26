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
        form = EditProfileForm.new(form_params.merge(user_record: current_user_record!))

        if form.invalid?
          return render(
            Settings::Profiles::ShowView.new(
              current_user: current_user!,
              form:
            ),
            status: :unprocessable_entity
          )
        end

        UpdateProfileService.new.call(
          user_record: current_user_record!,
          atname: form.atname.not_nil!,
          name: form.name.not_nil!,
          description: form.description.not_nil!
        )

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
