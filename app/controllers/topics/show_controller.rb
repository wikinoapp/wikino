# typed: true
# frozen_string_literal: true

module Topics
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record&.space_member_record(space_record:)
      space_member_policy = SpacePolicyFactory.build(
        user_record: current_user_record,
        space_member_record:
      )
      topic_record = space_member_policy.showable_topics(space_record:).find_by!(
        number: params[:topic_number]
      )

      pinned_page_records = topic_record.page_records.active.pinned
        .preload(featured_image_attachment_record: {active_storage_attachment_record: :blob})
        .order(pinned_at: :desc, id: :desc)
      pinned_pages = PageRepository.new.to_models(page_records: pinned_page_records, current_space_member: space_member_record)

      cursor_paginate_page = topic_record.not_nil!.page_records.active.not_pinned
        .preload(featured_image_attachment_record: {active_storage_attachment_record: :blob})
        .cursor_paginate(
          after: params[:after].presence,
          before: params[:before].presence,
          limit: 100,
          order: {modified_at: :desc, id: :desc}
        ).fetch
      pages = PageRepository.new.to_models(
        page_records: cursor_paginate_page.records,
        current_space_member: space_member_record
      )
      pagination = PaginationRepository.new.to_model(cursor_paginate_page:)
      page_list = PageList.new(pages:, pagination:)
      topic = TopicRepository.new.to_model(
        topic_record:,
        can_update: space_member_policy.can_update_topic?(topic_record:),
        can_create_page: space_member_policy.can_create_page?(topic_record:)
      )

      render_component Topics::ShowView.new(current_user:, topic:, pinned_pages:, page_list:)
    end
  end
end
