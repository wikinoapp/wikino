# typed: strict
# frozen_string_literal: true

module Types
  module Enums
    class UpsertNoteErrorCode < Types::Enums::Base
      value "INVALID_ERROR"
      value "DUPLICATED_NOTE_ERROR"
    end
  end
end
