# typed: true
# frozen_string_literal: true

module DraftPages
  class UpdateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      space_viewer = Current.viewer!.space_viewer!(space:)
      page = space.find_page_by_number!(params[:page_number]&.to_i)

      unless space_viewer.can_update_draft_page?(page:)
        return render_404
      end

      result = UpdateDraftPageService.new.call(
        space_member: T.let(space_viewer, SpaceMemberRecord),
        page:,
        topic_number: form_params[:topic_number],
        title: form_params[:title],
        body: form_params[:body]
      )
      draft_page_entity = result.draft_page.to_entity(space_viewer:)
      link_list_entity = result.draft_page.not_nil!.fetch_link_list_entity(space_viewer:)
      backlink_list_entity = page.not_nil!.fetch_backlink_list_entity(space_viewer:)

      render(DraftPages::UpdateView.new(
        draft_page_entity:,
        link_list_entity:,
        backlink_list_entity:
      ), {
        content_type: "text/vnd.turbo-stream.html",
        layout: false
      })
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:edit_page_form), ActionController::Parameters).permit(
        :topic_number,
        :title,
        :body
      )
    end
  end
end
