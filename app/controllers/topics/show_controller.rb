# typed: true
# frozen_string_literal: true

module Topics
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      space_viewer = Current.viewer!.space_viewer!(space:)
      topic = space_viewer.showable_topics.find_by!(number: params[:topic_number])

      pinned_pages = topic.page_records.active.pinned.order(pinned_at: :desc, id: :desc)

      cursor_paginate_page = topic.not_nil!.page_records.active.not_pinned.cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 100,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      page_entities = PageRecord.to_entities(space_viewer:, pages: cursor_paginate_page.records)
      pagination_entity = PaginationEntity.from_cursor_paginate(cursor_paginate_page:)

      render Topics::ShowView.new(
        current_user: current_user!,
        topic_entity: topic.to_entity(space_viewer:),
        pinned_page_entities: pinned_pages.map { _1.to_entity(space_viewer:) },
        page_list_entity: PageListEntity.new(page_entities:, pagination_entity:)
      )
    end
  end
end
