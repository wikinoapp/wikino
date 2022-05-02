# frozen_string_literal: true

module Types
  module Objects
    class QueryType < Types::Objects::Base
      include GraphQL::Types::Relay::HasNodeField
      include GraphQL::Types::Relay::HasNodesField

      field :viewer, Types::Objects::UserType, null: true

      def viewer
        context[:viewer]
      end
    end
  end
end
