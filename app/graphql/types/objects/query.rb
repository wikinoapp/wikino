# typed: true
# frozen_string_literal: true

module Types
  module Objects
    class Query < Types::Objects::Base
      include GraphQL::Types::Relay::HasNodeField
      include GraphQL::Types::Relay::HasNodesField

      field :viewer, Types::Objects::User, null: true

      def viewer
        context[:viewer]
      end
    end
  end
end
