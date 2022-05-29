# typed: strict
# frozen_string_literal: true

module NoteUpsertable
  extend T::Sig

  class Error < T::Struct
    const :code, String
    const :message, String
    const :original_note, T.nilable(Note)
  end

  class Result < T::Struct
    const :note, T.nilable(Note)
    const :errors, T::Array[Error]
  end
end
