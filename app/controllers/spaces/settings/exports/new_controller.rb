# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      class NewController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_member_record = current_user!.space_member_record(space_record:)
          space_entity = space.to_entity(space_viewer:)

          unless space_entity.viewer_can_export?
            return render_404
          end

          render Spaces::Settings::Exports::NewView.new(
            current_user: current_user!,
            space_entity:
          )
        end
      end
    end
  end
end
