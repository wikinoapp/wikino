# frozen_string_literal: true

module Types
  module Object
    class MutationErrorType < Types::Object::Base
      field :message, String, null: false
    end
  end
end
