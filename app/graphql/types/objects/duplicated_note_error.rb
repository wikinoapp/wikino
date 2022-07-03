# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class DuplicatedNoteError < Types::Objects::Base
      description "A error when is occurred that a title of updated note is duplicated."

      field :message, String, "A error message.", null: false
      field :original_note, Types::Objects::Note, "An original note.", null: false
    end
  end
end
