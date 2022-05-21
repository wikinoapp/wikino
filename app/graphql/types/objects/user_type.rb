# frozen_string_literal: true

module Types
  module Objects
    class UserType < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      field :note, Types::Objects::NoteType, null: true do
        argument :database_id, String, required: true
      end

      field :notes, Types::Objects::NoteType.connection_type, null: false do
        argument :q, String, required: false
        argument :order_by, Types::InputObjects::NoteOrder, required: true
      end

      def note(database_id:)
        object.notes.find(database_id)
      end

      def notes(q: "", order_by:)
        order = OrderProperty.build(order_by)

        notes = object.notes.where.not(modified_at: nil)

        if q.present?
          notes = notes.where("title like ?", "%#{q}%")
        end

        notes.order(order.field => order.direction)
      end
    end
  end
end
