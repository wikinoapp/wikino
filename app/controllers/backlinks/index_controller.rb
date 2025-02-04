# typed: true
# frozen_string_literal: true

module Backlinks
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

      unless Current.viewer.can_view_page?(page:)
        render_404
        return
      end

      backlink_collection = page.not_nil!.fetch_backlink_collection(after: params[:after])

      render(Backlinks::IndexView.new(backlink_collection:), {
        content_type: "text/vnd.turbo-stream.html",
        layout: false
      })
    end
  end
end
