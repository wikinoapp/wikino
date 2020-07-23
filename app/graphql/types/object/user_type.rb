# frozen_string_literal: true

module Types
  module Object
    class UserType < Types::Object::Base
      implements GraphQL::Types::Relay::Node

      field :notes, Types::Object::NoteType.connection_type, null: false do
        argument :order_by, Types::InputObject::NoteOrder, required: true
      end

      def notes(order_by:)
        order = OrderProperty.build(order_by)
        object.notes.order(order.field => order.direction)
      end
    end
  end
end
