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

      @form = EditNoteForm.new(
        list_number: @note.list.number,
        title: @note.title&.value,
        body: @note.body
      )
      @viewable_lists = viewer!.viewable_lists
    end
  end
end
