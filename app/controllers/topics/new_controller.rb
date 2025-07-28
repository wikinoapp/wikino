# typed: true
# frozen_string_literal: true

module Topics
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

      unless space_member_policy.can_create_topic?
        return render_404
      end

      space = SpaceRepository.new.to_model(space_record:)
      form = Topics::CreationForm.new

      render_component Topics::NewView.new(current_user: current_user!, space:, form:)
    end
  end
end
