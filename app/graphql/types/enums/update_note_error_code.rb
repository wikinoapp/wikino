# typed: strict
# frozen_string_literal: true

module Types
  module Enums
    class UpdateNoteErrorCode < Types::Enums::Base
      value "DUPLICATED", "Duplicated"
      value "INVALID", "Invalid"
    end
  end
end
