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
        space_entity = Space.find_by_identifier!(params[:space_identifier]).to_entity(viewer: Current.viewer!)

        unless space_entity.viewer_can_update?
          return render_404
        end

        render Spaces::Settings::ShowView.new(space_entity:)
      end
    end
  end
end
