# typed: true
# frozen_string_literal: true

module Links
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    layout false

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space = Space.find_by_identifier!(params[:space_identifier])
      space_viewer = Current.viewer!.space_viewer!(space:)
      page = space.find_page_by_number!(params[:page_number]&.to_i)

      unless space_viewer.can_view_page?(page:)
        return render_404
      end

      draft_page = space_viewer.draft_pages.find_by(page:)
      pageable = draft_page.presence || page

      link_list_entity = pageable.fetch_link_list_entity(space_viewer:, after: params[:after])

      render(Links::IndexView.new(page_entity: page.to_entity(space_viewer:), link_list_entity:), {
        content_type: "text/vnd.turbo-stream.html",
        layout: false
      })
    end
  end
end
