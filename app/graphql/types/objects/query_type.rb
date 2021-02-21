# frozen_string_literal: true

module Types
  module Objects
    class QueryType < Types::Objects::Base
      add_field GraphQL::Types::Relay::NodeField
      add_field GraphQL::Types::Relay::NodesField

      field :viewer, Types::Objects::UserType, null: true

      def viewer
        context[:viewer]
      end
    end
  end
end
