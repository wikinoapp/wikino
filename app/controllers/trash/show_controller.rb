# typed: true
# frozen_string_literal: true

module Trash
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record!.space_member_record(space_record:)
      space_member_policy = SpacePolicyFactory.build(
        user_record: current_user_record!,
        space_member_record:
      )

      unless space_member_policy.can_show_trash?(space_record:)
        return render_404
      end

      space = SpaceRepository.new.to_model(space_record:)
      page_list = PageListRepository.new.restorable(
        space_record:,
        before: params[:before],
        after: params[:after]
      )
      form = Pages::BulkRestoringForm.new

      render_component Trash::ShowView.new(
        current_user: current_user!,
        space:,
        page_list:,
        form:
      )
    end
  end
end
