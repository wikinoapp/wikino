# typed: true
# frozen_string_literal: true

module Backlinks
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware

    layout false

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record&.space_member_record(space_record:)
      page_record = space_record.page_record_by_number!(params[:page_number])
      topic_policy = topic_policy_for(topic_record: page_record.topic_record.not_nil!)

      unless topic_policy.can_show_page?(page_record:)
        return render_404
      end

      backlink_list = BacklinkListRepository.new.to_model(
        user_record: current_user_record,
        page_record:,
        after: params[:after]
      )
      page = PageRepository.new.to_model(page_record:, current_space_member: space_member_record)

      render_component(
        Backlinks::IndexView.new(page:, backlink_list:),
        content_type: "text/vnd.turbo-stream.html",
        layout: false
      )
    end
  end
end
