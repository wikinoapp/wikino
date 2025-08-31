# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Deletions
      class NewController < ApplicationController
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

          space = SpaceRepository.new.to_model(space_record:)
          form = Spaces::DestroyConfirmationForm.new(space_record:)

          render_component Spaces::Settings::Deletions::NewView.new(
            current_user: current_user!,
            space:,
            form:
          )
        end
      end
    end
  end
end
