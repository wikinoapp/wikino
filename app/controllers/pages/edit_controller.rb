# typed: true
# frozen_string_literal: true

module Pages
  class EditController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      page = space.find_page_by_number!(params[:page_number]&.to_i)

      unless Current.viewer!.can_update_page?(page:)
        return render_404
      end

      draft_page = Current.viewer!.active_draft_pages.find_by(page:)
      pageable = draft_page.presence || page

      form = EditPageForm.new(
        topic_number: pageable.topic.number,
        title: pageable.title,
        body: pageable.body
      )

      link_collection = pageable.fetch_link_collection
      backlink_collection = page.not_nil!.fetch_backlink_collection

      render(Pages::EditView.new(space:, page:, draft_page:, form:, link_collection:, backlink_collection:))
    end
  end
end
