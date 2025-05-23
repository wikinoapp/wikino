# typed: true
# frozen_string_literal: true

module Pages
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record&.space_member_record(space_record:)
      page_record = space_record.find_page_by_number!(params[:page_number]&.to_i)
      space_member_policy = SpaceMemberPolicy.new(
        user_record: current_user_record,
        space_member_record:
      )

      unless space_member_policy.can_show_page?(page_record:)
        return render_404
      end

      page = PageRepository.new.to_model(
        page_record:,
        can_update: space_member_policy.can_update_page?(page_record:)
      )
      link_list = LinkListRepository.new.to_model(
        user_record: current_user_record,
        pageable_record: page_record
      )
      backlink_list = BacklinkListRepository.new.to_model(
        user_record: current_user_record,
        page_record:
      )

      render_component Pages::ShowView.new(
        current_user:,
        page:,
        link_list:,
        backlink_list:
      )
    end
  end
end
