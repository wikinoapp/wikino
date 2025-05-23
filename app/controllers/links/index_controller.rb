# typed: true
# frozen_string_literal: true

module Links
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    layout false

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

      draft_page_record = space_member_record&.draft_page_records&.find_by(page_record:)
      pageable_record = draft_page_record.presence || page_record

      page = PageRepository.new.to_model(page_record:)
      link_list = LinkListRepository.new.to_model(
        user_record: current_user_record,
        pageable_record:,
        after: params[:after]
      )

      render_component(
        Links::IndexView.new(page:, link_list:),
        content_type: "text/vnd.turbo-stream.html",
        layout: false
      )
    end
  end
end
