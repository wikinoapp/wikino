# typed: true
# frozen_string_literal: true

module Pages
  class EditController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = Space.find_by_identifier!(params[:space_identifier])
      space_viewer = Current.viewer!.space_viewer!(space:)
      page = space.find_page_by_number!(params[:page_number]&.to_i)

      unless space_viewer.can_update_page?(page:)
        return render_404
      end

      space_member = T.let(space_viewer, SpaceMember)
      draft_page = space_member.draft_pages.find_by(page:)
      pageable = draft_page.presence || page

      form = EditPageForm.new(
        space_member:,
        topic_number: pageable.topic.not_nil!.number,
        title: pageable.title,
        body: pageable.body
      )

      link_list_entity = pageable.not_nil!.fetch_link_list_entity(space_viewer:)
      backlink_list_entity = page.not_nil!.fetch_backlink_list_entity(space_viewer:)

      render Pages::EditView.new(
        space_entity: space.to_entity(space_viewer:),
        page_entity: page.to_entity(space_viewer:),
        draft_page_entity: draft_page&.to_entity(space_viewer:),
        form:,
        link_list_entity:,
        backlink_list_entity:
      )
    end
  end
end
