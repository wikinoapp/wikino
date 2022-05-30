# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class DuplicatedNoteError < Types::Objects::Base
      field :message, String, null: false
      field :original_note, Types::Objects::Note, null: false
    end
  end
end
