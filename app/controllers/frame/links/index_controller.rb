# typed: true
# frozen_string_literal: true

module Frame
  module Links
    class IndexController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Authorizable
      include ControllerConcerns::Localizable
      include ControllerConcerns::PageSettable

      around_action :set_locale
      before_action :require_authentication
      before_action :set_page

      sig { returns(T.untyped) }
      def call
        authorize(@page, :show?)

        draft_page = viewer!.draft_pages.find_by(page: @page)
        page_editable = draft_page.presence || @page

        @link_collection = page_editable.fetch_link_collection(after: params[:after])

        render(content_type: "text/vnd.turbo-stream.html", layout: false)
      end
    end
  end
end
