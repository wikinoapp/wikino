# typed: true
# frozen_string_literal: true

module Home
  class ShowController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::Authorizable

    around_action :set_locale
    before_action :set_current_space

    sig { returns(T.untyped) }
    def call
      restore_session

      @pinned_pages = viewable_pages.pinned.order(pinned_at: :desc, id: :desc)

      cursor_paginate_page = viewable_pages.not_pinned.cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 100,
        order: {modified_at: :desc, id: :desc}
      ).fetch
      @pages = cursor_paginate_page.records
      @pagination = Pagination.from_cursor_paginate(cursor_paginate_page:)
    end

    sig { returns(Page::PrivateAssociationRelation) }
    private def viewable_pages
      pages = Current.space!.pages.published.joins(:topic)

      if signed_in? && Current.space! == Current.user!.space
        pages.merge(Current.user!.topics)
      else
        pages.merge(Topic.visibility_public)
      end
    end
  end
end
