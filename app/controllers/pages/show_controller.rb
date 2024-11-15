# typed: true
# frozen_string_literal: true

module Pages
  class ShowController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::PageSettable

    around_action :set_locale
    before_action :set_current_space
    before_action :restore_session
    before_action :set_page

    rescue_from Pundit::NotAuthorizedError, with: :render_404

    sig { returns(T.untyped) }
    def call
      authorize(@page, :show?)

      @link_collection = T.let(@page.not_nil!.fetch_link_collection, T.nilable(LinkCollection))
      @backlink_collection = T.let(@page.not_nil!.fetch_backlink_collection, T.nilable(BacklinkCollection))
    end
  end
end
