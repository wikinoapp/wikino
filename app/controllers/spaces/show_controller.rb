# typed: true
# frozen_string_literal: true

module Spaces
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record&.space_member_record(space_record:)
      space_member_policy = SpaceMemberPolicyFactory.build(
        user_record: current_user_record,
        space_member_record:
      )
      showable_pages = space_member_policy.showable_pages(space_record:).preload(:topic_record)

      cursor_paginate_page = showable_pages.not_pinned.cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 100,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      pages = PageRepository.new.to_models(page_records: cursor_paginate_page.records, current_space_member: space_member_record)
      pagination = PaginationRepository.new.to_model(cursor_paginate_page:)
      page_list = PageList.new(pages:, pagination:)

      space = SpaceRepository.new.to_model(
        space_record:,
        can_create_topic: space_member_policy.can_create_topic?
      )

      first_joined_topic_record = space_member_policy.joined_topic_records.order(:id).first
      first_joined_topic = if first_joined_topic_record
        TopicRepository.new.to_model(topic_record: first_joined_topic_record)
      end

      pinned_page_records = showable_pages.pinned.order(pinned_at: :desc, id: :desc)
      pinned_pages = PageRepository.new.to_models(page_records: pinned_page_records, current_space_member: space_member_record)

      render_component Spaces::ShowView.new(
        current_user:,
        joined_space: space_member_record.present?,
        space:,
        first_joined_topic:,
        pinned_pages:,
        page_list:
      )
    end
  end
end
