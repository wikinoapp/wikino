# typed: true
# frozen_string_literal: true

module Links
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    layout false

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      page = space.find_page_by_number!(params[:page_number]&.to_i)

      unless Current.viewer!.can_view_page?(page:)
        return render_404
      end

      draft_page = Current.viewer!.active_draft_pages.find_by(page:)
      pageable = draft_page.presence || page

      link_collection = pageable.fetch_link_collection(after: params[:after])

      render(Links::IndexView.new(link_collection:), {
        content_type: "text/vnd.turbo-stream.html",
        layout: false
      })
    end
  end
end
