# frozen_string_literal: true

module Types
  module Objects
    class MutationErrorType < Types::Objects::Base
      field :message, String, null: false
    end
  end
end
