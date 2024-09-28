# typed: true
# frozen_string_literal: true

module Notes
  class EditController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::NoteSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_note

    sig { returns(T.untyped) }
    def call
      authorize(@note, :edit?)

      @draft_note = viewer!.draft_notes.find_by(note: @note)
      note_editable = @draft_note.presence || @note

      @form = EditNoteForm.new(
        viewer: viewer!,
        notebook_number: note_editable.notebook.number,
        title: note_editable.title,
        body: note_editable.body
      )

      @link_list = note_editable.fetch_link_list
      @backlink_list = note_editable.fetch_backlink_list
    end
  end
end
