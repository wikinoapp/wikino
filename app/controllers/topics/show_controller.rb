# typed: true
# frozen_string_literal: true

module Topics
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :restore_user_session

    rescue_from Pundit::NotAuthorizedError, with: :render_404

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      topic = Current.viewer!.find_topic_by_number!(space:, number: params[:topic_number])

      pinned_pages = topic.pages.active.pinned.order(pinned_at: :desc, id: :desc)

      cursor_paginate_page = topic.not_nil!.pages.active.not_pinned.cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 100,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      pages = cursor_paginate_page.records
      pagination = Pagination.from_cursor_paginate(cursor_paginate_page:)

      render Topics::ShowView.new(
        topic:,
        pinned_pages:,
        page_connection: PageConnection.new(pages:, pagination:)
      )
    end
  end
end
