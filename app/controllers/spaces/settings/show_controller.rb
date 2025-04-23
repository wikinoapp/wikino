# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    class ShowController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
        space_member_record = current_user!.space_member_record(space_record:)
        space_entity = space.to_entity(space_viewer:)

        unless space_entity.viewer_can_update?
          return render_404
        end

        render Spaces::Settings::ShowView.new(
          current_user: current_user!,
          space_entity:
        )
      end
    end
  end
end
