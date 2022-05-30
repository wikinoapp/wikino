# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class MutationError < Types::Objects::Base
      field :message, String, null: false
    end
  end
end
