# typed: true
# frozen_string_literal: true

module DraftPages
  class UpdateController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @page = Current.space!.find_pages_by_number!(params[:page_number]&.to_i)
      authorize(@page, :update?)

      result = UpdateDraftPageUseCase.new.call(
        page: @page.not_nil!,
        topic_number: form_params[:topic_number],
        title: form_params[:title],
        body: form_params[:body]
      )
      @draft_page = result.draft_page
      @link_collection = @draft_page.fetch_link_collection
      @backlink_collection = @draft_page.page.not_nil!.fetch_backlink_collection

      render(content_type: "text/vnd.turbo-stream.html", layout: false)
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
