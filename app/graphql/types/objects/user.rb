# typed: true
# frozen_string_literal: true

module Types
  module Objects
    class User < Types::Objects::Base
      implements GraphQL::Types::Relay::Node

      field :note, Types::Objects::Note, null: true do
        argument :database_id, String, required: true
      end

      field :notes, Types::Objects::Note.connection_type, null: false do
        argument :order_by, Types::InputObjects::NoteOrder, required: true
        argument :q, String, required: false
      end

      def note(database_id:)
        object.notes.find(database_id)
      end

      def notes(order_by:, q: "")
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
