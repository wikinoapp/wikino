# typed: true
# frozen_string_literal: true

module Notes
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::NotebookSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_notebook

    sig { returns(T.untyped) }
    def call
      authorize(Note.new, :new?)

      result = CreateInitialNoteUseCase.new.call(notebook: @notebook.not_nil!, viewer: viewer!)

      redirect_to edit_note_path(viewer!.space_identifier, result.note.number)
    end
  end
end
