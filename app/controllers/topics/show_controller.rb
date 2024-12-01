# typed: true
# frozen_string_literal: true

module Topics
  class ShowController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :set_current_space
    before_action :restore_session

    rescue_from Pundit::NotAuthorizedError, with: :render_404

    sig { returns(T.untyped) }
    def call
      @topic = Current.space!.topics.kept.find_by!(number: params[:topic_number])
      authorize(@topic, :show?)

      @pinned_pages = @topic.pages.active.pinned.order(pinned_at: :desc, id: :desc)

      cursor_paginate_page = @topic.not_nil!.pages.active.not_pinned.cursor_paginate(
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
