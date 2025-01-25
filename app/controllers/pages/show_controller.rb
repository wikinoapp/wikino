# typed: true
# frozen_string_literal: true

module Pages
  class ShowController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :set_current_space
    before_action :restore_user_session

    rescue_from Pundit::NotAuthorizedError, with: :render_404

    sig { returns(T.untyped) }
    def call
      page = Current.space!.find_pages_by_number!(params[:page_number]&.to_i)

      authorize(page, :show?)

      link_collection = page.not_nil!.fetch_link_collection
      backlink_collection = page.not_nil!.fetch_backlink_collection

      render Pages::ShowView.new(page:, link_collection:, backlink_collection:)
    end
  end
end
