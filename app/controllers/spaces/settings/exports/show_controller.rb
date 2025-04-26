# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Exports
      class ShowController < ApplicationController
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

          unless space_member_policy.can_export_space?(space_record:)
            return render_404
          end

          export_record = space_record.export_records.find(params[:export_id])
          space = SpaceRepository.new.to_model(space_record:)
          export = ExportRepository.new.to_model(export_record:)
          export_status = ExportStatusRepository.new.to_model(
            export_status_record: export_record.latest_status_record
          )

          render Spaces::Settings::Exports::ShowView.new(
            current_user:,
            space:,
            export:,
            export_status:
          )
        end
      end
    end
  end
end
