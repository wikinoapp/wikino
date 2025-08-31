# typed: true
# frozen_string_literal: true

module Pages
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      page_record = current_space_record.page_record_by_number!(params[:page_number])
        .tap { |p| p.featured_image_attachment_record&.active_storage_attachment_record }
      space_member_record = current_space_member_record(space_record: current_space_record)
      topic_policy = topic_policy_for(topic_record: page_record.topic_record.not_nil!)

      unless topic_policy.can_show_page?(page_record:)
        return render_404
      end

      page = PageRepository.new.to_model(
        page_record:,
        can_update: topic_policy.can_update_page?(page_record:),
        current_space_member: space_member_record
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
