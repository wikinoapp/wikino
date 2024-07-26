# typed: strict
# frozen_string_literal: true

module NoteUpsertable
  #   extend T::Sig
  #
  #   class Error < T::Struct
  #     const :message, String
  #   end
  #
  #   class DuplicatedNoteError < T::Struct
  #     const :message, String
  #     const :original_note, T.nilable(Note)
  #   end
  #
  #   class Result < T::Struct
  #     const :note, T.nilable(Note)
  #     const :errors, T::Array[T.any(Error, DuplicatedNoteError)]
  #   end
  #
  #   sig { params(form: T.any(NoteCreatingForm, NoteUpdatingForm)).returns(T::Array[T.any(Error, DuplicatedNoteError)]) }
  #   def errors_from_form(form)
  #     form.errors.map do |error|
  #       if error.attribute == :title && error.type == :title_should_be_unique
  #         DuplicatedNoteError.new(message: error.full_message, original_note: form.original_note)
  #       else
  #         Error.new(message: error.full_message)
  #       end
  #     end
  #   end
end
