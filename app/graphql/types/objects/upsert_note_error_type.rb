# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class UpsertNoteErrorType < Types::Objects::Base
      field :code, Types::Enums::UpsertNoteErrorCode, null: false
      field :message, String, null: false
      field :original_note, Types::Objects::NoteType, null: true
    end
  end
end
