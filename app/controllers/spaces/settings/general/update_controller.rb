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
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_member_record = current_user_record!.space_member_record(space_record:)
          space_member_policy = SpaceMemberPolicy.new(
            user_record: current_user_record!,
            space_member_record:
          )

          unless space_member_policy.can_update_space?(space_record:)
            return render_404
          end

          form = SpaceForm::Edit.new(form_params.merge(space_record:))

          if form.invalid?
            space = SpaceRepository.new.to_model(space_record:)

            return render(Spaces::Settings::General::ShowView.new(
              current_user: current_user!,
              space:,
              form:
            ), {
              status: :unprocessable_entity
            })
          end

          SpaceService::Update.new.call(
            space_record:,
            identifier: form.identifier.not_nil!,
            name: form.name.not_nil!
          )

          flash[:notice] = t("messages.spaces.updated")
          redirect_to space_settings_general_path(space_record.identifier)
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
