# typed: true
# frozen_string_literal: true

module Spaces
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceAware

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space_record = current_space_record
      space_member_record = current_space_member_record(space_record:)
      space_policy = space_policy_for(space_record:)
      showable_pages = space_policy.showable_pages(space_record:).preload(:topic_record)

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
        can_create_topic: space_policy.can_create_topic?
      )

      first_joined_topic_record = space_policy.joined_topic_records.order(:id).first
      first_joined_topic = if first_joined_topic_record
        TopicRepository.new.to_model(topic_record: first_joined_topic_record)
      end

      pinned_page_records = showable_pages.pinned.order(pinned_at: :desc, id: :desc)
      pinned_pages = PageRepository.new.to_models(page_records: pinned_page_records, current_space_member: space_member_record)

      # トピック一覧の取得
      topic_repository = TopicRepository.new
      topics = if space_member_record.present?
        # スペースメンバーの場合は参加トピックを取得
        topic_repository.find_topics_by_space(space_record:, space_member_record:)
      else
        # ゲストの場合は公開トピックを取得
        topic_repository.find_public_topics_by_space(space_record:)
      end

      render_component Spaces::ShowView.new(
        current_user:,
        joined_space: space_member_record.present?,
        space:,
        first_joined_topic:,
        pinned_pages:,
        page_list:,
        topics:
      )
    end
  end
end
