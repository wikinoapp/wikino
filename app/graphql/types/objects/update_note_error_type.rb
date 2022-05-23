# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class UpdateNoteErrorType < Types::Objects::Base
      field :code, Types::Enums::UpdateNoteErrorCode, null: false
      field :original_note, Types::Objects::NoteType, null: true
    end
  end
end
