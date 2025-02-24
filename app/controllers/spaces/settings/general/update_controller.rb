# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module General
      class UpdateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space = Space.find_by_identifier!(params[:space_identifier])
          space_viewer = Current.viewer!.space_viewer!(space:)
          space_entity = space.to_entity(space_viewer:)

          unless space_entity.viewer_can_update?
            return render_404
          end

          form = EditSpaceForm.new(form_params.merge(space:))

          if form.invalid?
            return render(
              Spaces::Settings::General::ShowView.new(space_entity:, form:),
              status: :unprocessable_entity
            )
          end

          UpdateSpaceUseCase.new.call(space:, form:)

          flash[:notice] = t("messages.spaces.updated")
          redirect_to space_settings_general_path(space.identifier)
        end

        sig { returns(ActionController::Parameters) }
        private def form_params
          T.cast(params.require(:edit_space_form), ActionController::Parameters).permit(
            :identifier,
            :name
          )
        end
      end
    end
  end
end
