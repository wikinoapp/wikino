# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class MutationErrorType < Types::Objects::Base
      field :message, String, null: false
      field :code, Types::Enums::MutationErrorCode, null: true
    end
  end
end
