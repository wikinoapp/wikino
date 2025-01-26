# typed: true
# frozen_string_literal: true

module Pages
  class EditController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @page = Current.space!.find_page_by_number!(params[:page_number]&.to_i)
      authorize(@page, :edit?)

      @draft_page = Current.user!.draft_pages.find_by(page: @page)
      pageable = @draft_page.presence || @page

      @form = EditPageForm.new(
        topic_number: pageable.topic.number,
        title: pageable.title,
        body: pageable.body
      )

      @link_collection = pageable.fetch_link_collection
      @backlink_collection = @page.not_nil!.fetch_backlink_collection
    end
  end
end
