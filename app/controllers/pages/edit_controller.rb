# typed: true
# frozen_string_literal: true

module Pages
  class EditController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::PageSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_page

    sig { returns(T.untyped) }
    def call
      authorize(@page, :edit?)

      @draft_page = viewer!.draft_pages.find_by(page: @page)
      pageable = @draft_page.presence || @page

      @form = EditPageForm.new(
        viewer: viewer!,
        topic_number: pageable.topic.number,
        title: pageable.title,
        body: pageable.body
      )

      @link_collection = pageable.fetch_link_collection
      @backlink_collection = @page.not_nil!.fetch_backlink_collection
    end
  end
end
