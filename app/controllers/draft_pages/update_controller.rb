# typed: true
# frozen_string_literal: true

module DraftPages
  class UpdateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      page = space.find_page_by_number!(params[:page_number]&.to_i)

      unless Current.viewer!.can_update_draft_page?(page:)
        return render_404
      end

      result = UpdateDraftPageUseCase.new.call(
        page:,
        topic_number: form_params[:topic_number],
        title: form_params[:title],
        body: form_params[:body]
      )
      draft_page = result.draft_page
      link_collection = draft_page.fetch_link_collection
      backlink_collection = draft_page.page.not_nil!.fetch_backlink_collection

      render(DraftPages::UpdateView.new(draft_page:, link_collection:, backlink_collection:), {
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
