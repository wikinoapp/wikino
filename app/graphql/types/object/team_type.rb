# frozen_string_literal: true

module Types
  module Object
    class TeamType < Types::Object::Base
      implements GraphQL::Types::Relay::Node

      field :teamname, String, null: false
      field :name, String, null: false

      field :note, Types::Object::NoteType, null: true do
        argument :note_number, GraphQL::Types::BigInt, required: true
      end

      field :notes, Types::Object::NoteType.connection_type, null: false do
        argument :order_by, Types::InputObject::NoteOrder, required: true
      end

      def note(note_number:)
        object.notes.find_by(number: note_number)
      end

      def notes(order_by:)
        order = OrderProperty.build(order_by)
        object.notes.order(order.field => order.direction)
      end
    end
  end
end
