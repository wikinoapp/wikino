# typed: true
# frozen_string_literal: true

module Links
  class IndexController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable

    layout false

    around_action :set_locale
    before_action :set_current_space
    before_action :restore_session

    rescue_from Pundit::NotAuthorizedError, with: :render_404

    sig { returns(T.untyped) }
    def call
      @page = Current.space!.find_pages_by_number!(params[:page_number]&.to_i)
      authorize(@page, :show?)

      draft_page = Current.user&.draft_pages&.find_by(page: @page)
      pageable = draft_page.presence || @page

      @link_collection = pageable.fetch_link_collection(after: params[:after])

      render(content_type: "text/vnd.turbo-stream.html", layout: false)
    end
  end
end
