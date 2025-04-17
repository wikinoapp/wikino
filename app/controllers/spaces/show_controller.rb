# typed: true
# frozen_string_literal: true

module Spaces
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_viewer = Current.viewer!.space_viewer!(space:)
      showable_pages = space_viewer.showable_pages.preload(:topic_record)

      cursor_paginate_page = showable_pages.not_pinned.cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 100,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      page_entities = PageRecord.to_entities(space_viewer:, pages: cursor_paginate_page.records)
      pagination_entity = PaginationEntity.from_cursor_paginate(cursor_paginate_page:)

      space_entity = space.to_entity(space_viewer:)
      first_topic_entity = space_viewer.joined_topics.first&.to_entity(space_viewer:)
      pinned_page_entities = PageRecord.to_entities(space_viewer:, pages: showable_pages.pinned.order(pinned_at: :desc, id: :desc))

      render Spaces::ShowView.new(
        current_user_entity: Current.viewer!.user_entity,
        space_entity:,
        first_topic_entity:,
        pinned_page_entities:,
        page_list_entity: PageListEntity.new(page_entities:, pagination_entity:)
      )
    end
  end
end
