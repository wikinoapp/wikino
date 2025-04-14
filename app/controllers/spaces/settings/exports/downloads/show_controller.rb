# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      module Downloads
        class ShowController < ApplicationController
          include ControllerConcerns::Authenticatable
          include ControllerConcerns::Localizable

          around_action :set_locale
          before_action :require_authentication

          sig { returns(T.untyped) }
          def call
            space = SpaceRecord.find_by_identifier!(params[:space_identifier])
            space_viewer = Current.viewer!.space_viewer!(space:)
            space_entity = space.to_entity(space_viewer:)

            unless space_entity.viewer_can_export?
              return render_404
            end

            export = space.exports.find(params[:export_id])

            unless export.active?
              return render_404
            end

            redirect_to(export.presigned_url, allow_other_host: true)
          end
        end
      end
    end
  end
end
