# frozen_string_literal: true

module Types
  module Object
    class QueryType < Types::Object::Base
      add_field GraphQL::Types::Relay::NodeField
      add_field GraphQL::Types::Relay::NodesField

      field :viewer, Types::Object::TeamMemberType, null: false

      def viewer
        context[:viewer]
      end
    end
  end
end
