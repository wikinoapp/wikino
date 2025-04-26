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
          space_member_record = current_user_record!.space_member_record(space_record:)
          space_member_policy = SpaceMemberPolicy.new(
            user_record: current_user_record!,
            space_member_record:
          )

          unless space_member_policy.can_export_space?(space_record:)
            return render_404
          end

          space = SpaceRepository.new.to_model(space_record:)

          render Spaces::Settings::Exports::NewView.new(current_user:, space:)
        end
      end
    end
  end
end
