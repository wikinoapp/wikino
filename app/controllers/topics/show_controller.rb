# typed: true
# frozen_string_literal: true

module Topics
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_topic

    sig { returns(T.untyped) }
    def call
      cursor_paginate_page = @topic.not_nil!.pages.published.cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 100,
        order: {modified_at: :desc, id: :desc}
      ).fetch

      @pages = cursor_paginate_page.records
      @pagination = Pagination.from_cursor_paginate(cursor_paginate_page:)
    end
  end
end
