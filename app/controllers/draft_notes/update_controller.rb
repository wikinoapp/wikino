# typed: true
# frozen_string_literal: true

module DraftNotes
  class UpdateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::NoteSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_note

    sig { returns(T.untyped) }
    def call
      authorize(@note, :update?)

      result = UpdateDraftNoteUseCase.new.call(
        viewer: viewer!,
        note: @note.not_nil!,
        notebook_number: form_params[:notebook_number],
        title: form_params[:title],
        body: form_params[:body]
      )
      @draft_note = result.draft_note
      @link_list = @draft_note.fetch_link_list
      @backlink_list = @draft_note.fetch_backlink_list

      render(content_type: "text/vnd.turbo-stream.html", layout: false)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:edit_note_form), ActionController::Parameters).permit(
        :notebook_number,
        :title,
        :body
      )
    end
  end
end
