# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module General
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

          unless space_member_policy.can_update_space?(space_record:)
            return render_404
          end

          space = SpaceRepository.new.to_model(space_record:)
          form = Spaces::EditForm.new(
            identifier: space_record.identifier,
            name: space_record.name
          )

          render_component Spaces::Settings::General::ShowView.new(
            current_user: current_user!,
            space:,
            form:
          )
        end
      end
    end
  end
end
