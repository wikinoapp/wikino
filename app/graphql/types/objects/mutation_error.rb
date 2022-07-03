# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class MutationError < Types::Objects::Base
      description "A mutation error."

      field :message, String, "An error message.", null: false
    end
  end
end
