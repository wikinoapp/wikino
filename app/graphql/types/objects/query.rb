# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class Query < Types::Objects::Base
      extend T::Sig

      description "Querying operations."

      include GraphQL::Types::Relay::HasNodeField
      include GraphQL::Types::Relay::HasNodesField

      field :viewer, Types::Objects::User, "Fetches the authenticated user.", null: true

      sig { returns(T.nilable(::User)) }
      def viewer
        context[:viewer]
      end
    end
  end
end
