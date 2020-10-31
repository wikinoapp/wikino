# frozen_string_literal: true

module Types
  module Objects
    class UpdateNoteErrorType < Types::Objects::Base
      field :code, Types::Enums::UpdateNoteErrorCode, null: false
    end
  end
end
