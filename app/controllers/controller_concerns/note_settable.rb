# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module NoteSettable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { void }
    private def set_note
      @note = T.let(viewer!.space.notes.find_by!(number: params[:note_number]), T.nilable(Note))
    end
  end
end
