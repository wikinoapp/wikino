# typed: true
# frozen_string_literal: true

module Backlinks
  class IndexController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::PageSettable

    layout false

    around_action :set_locale
    before_action :set_current_space
    before_action :restore_session
    before_action :set_page

    rescue_from Pundit::NotAuthorizedError, with: :render_404

    sig { returns(T.untyped) }
    def call
      authorize(@page, :show?)

      @backlink_collection = @page.fetch_backlink_collection(after: params[:after])

      render(content_type: "text/vnd.turbo-stream.html", layout: false)
    end
  end
end
