# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module General
      class UpdateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable
        include ControllerConcerns::SpaceAware

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = current_space_record
          space_policy = space_policy_for(space_record:)

          unless space_policy.can_update_space?(space_record:)
            return render_404
          end

          form = Spaces::EditForm.new(form_params.merge(space_record:))

          if form.invalid?
            space = SpaceRepository.new.to_model(space_record:)

            return render_component(
              Spaces::Settings::General::ShowView.new(
                current_user: current_user!,
                space:,
                form:
              ),
              status: :unprocessable_entity
            )
          end

          Spaces::UpdateService.new.call(
            space_record:,
            identifier: form.identifier.not_nil!,
            name: form.name.not_nil!
          )

          flash[:notice] = t("messages.spaces.updated")
          redirect_to space_settings_general_path(space_record.identifier)
        end

        sig { returns(ActionController::Parameters) }
        private def form_params
          T.cast(params.require(:spaces_edit_form), ActionController::Parameters).permit(
            :identifier,
            :name
          )
        end
      end
    end
  end
end
