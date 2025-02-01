# typed: true
# frozen_string_literal: true

module Pages
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      page = space.find_page_by_number!(params[:page_number]&.to_i)

      unless Current.viewer!.can_view_page?(page:)
        return render_404
      end

      link_collection = page.not_nil!.fetch_link_collection
      backlink_collection = page.not_nil!.fetch_backlink_collection

      render Pages::ShowView.new(page:, link_collection:, backlink_collection:)
    end
  end
end
