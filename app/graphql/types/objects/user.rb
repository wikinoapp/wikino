# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class User < Types::Objects::Base
      extend T::Sig

      implements GraphQL::Types::Relay::Node

      field :note, Types::Objects::Note, null: true do
        argument :database_id, String, required: true
      end

      field :notes, Types::Objects::Note.connection_type, null: false do
        argument :order_by, Types::InputObjects::NoteOrder, required: true
        argument :q, String, required: false
      end

      sig { params(database_id: String).returns(::Note) }
      def note(database_id:)
        object.notes.find(database_id)
      end

      sig { params(order_by: Types::InputObjects::NoteOrder, q: String).returns(ActiveRecord::Relation) }
      def notes(order_by:, q: "")
        order = OrderProperty.build(order_by.to_h)

        notes = object.notes.where.not(modified_at: nil)

        if q.present?
          notes = notes.where("title like ?", "%#{q}%")
        end

        notes.order(order.field => order.direction)
      end
    end
  end
end
