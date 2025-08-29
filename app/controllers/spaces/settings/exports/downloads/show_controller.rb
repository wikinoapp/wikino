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
            space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
            space_member_record = current_user_record!.space_member_record(space_record:)
            space_member_policy = SpaceMemberPolicyFactory.build(
              user_record: current_user_record!,
              space_member_record:
            )

            unless space_member_policy.can_export_space?(space_record:)
              return render_404
            end

            export_record = space_record.export_records.find(params[:export_id])

            unless export_record.active?
              return render_404
            end

            redirect_to(export_record.presigned_url, allow_other_host: true)
          end
        end
      end
    end
  end
end
