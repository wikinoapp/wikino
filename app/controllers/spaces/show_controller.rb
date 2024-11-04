# typed: strict
# frozen_string_literal: true

module Spaces
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

      @pinned_pages = if signed_in?
        Current.space!.pages.published.pinned.order(pinned_at: :desc, id: :desc)
      else
        Current.space!.pages.published.pinned.joins(:topic).merge(Topic.visibility_public).order(pinned_at: :desc, id: :desc)
      end

      cursor_paginate_page = if signed_in?
        Current.space!.pages.published.not_pinned.cursor_paginate(
          after: params[:after].presence,
          before: params[:before].presence,
          limit: 100,
          order: {modified_at: :desc, id: :desc}
        ).fetch
      else
        Current.space!.pages.published.not_pinned.joins(:topic).merge(Topic.visibility_public).cursor_paginate(
          after: params[:after].presence,
          before: params[:before].presence,
          limit: 100,
          order: {modified_at: :desc, id: :desc}
        ).fetch
      end

      @pages = cursor_paginate_page.records
      @pagination = Pagination.from_cursor_paginate(cursor_paginate_page:)
    end
  end
end
