# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Deletions
      class CreateController < ApplicationController
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

          form = Spaces::DestroyConfirmationForm.new(form_params.merge(space_record:))

          if form.invalid?
            space = SpaceRepository.new.to_model(space_record:)

            return render(
              Spaces::Settings::Deletions::NewView.new(
                current_user: current_user!,
                space:,
                form:
              ),
              status: :unprocessable_entity
            )
          end

          Spaces::SoftDestroyService.new.call(space_record:)

          flash[:notice] = t("messages.spaces.deleted")
          redirect_to root_path
        end

        sig { returns(ActionController::Parameters) }
        private def form_params
          T.cast(params.require(:spaces_destroy_confirmation_form), ActionController::Parameters).permit(
            :space_name
          )
        end
      end
    end
  end
end
