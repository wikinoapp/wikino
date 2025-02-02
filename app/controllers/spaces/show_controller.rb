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
      space = find_space_by_identifier!
      space_viewer = Current.viewer!.space_viewer!(space:)
      joined_topics = space_viewer.topics
      first_joined_topic = joined_topics.first
      joined_topic_pages = space_viewer.viewable_pages.joins(:topic).merge(joined_topics)
      pinned_pages = joined_topic_pages.pinned.order(pinned_at: :desc, id: :desc)

      cursor_paginate_page = joined_topic_pages.not_pinned.cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 100,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      pages = cursor_paginate_page.records
      pagination = Pagination.from_cursor_paginate(cursor_paginate_page:)

      render Spaces::ShowView.new(
        space:,
        space_viewer:,
        first_joined_topic:,
        pinned_pages:,
        page_connection: PageConnection.new(pages:, pagination:)
      )
    end
  end
end
