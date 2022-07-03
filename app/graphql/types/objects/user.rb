# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class User < Types::Objects::Base
      extend T::Sig

      description "A user."

      implements GraphQL::Types::Relay::Node

      field :note, Types::Objects::Note, "Fetches a note associated with the user.", null: true do
        argument :database_id, String, "Identifies the primary key from the database.", required: true
      end

      field :notes, Types::Objects::Note.connection_type, "Fetches a list of notes associated with the user.", null: false do
        argument :order_by, Types::InputObjects::NoteOrder, "A note order.", required: true
        argument :q, String, "A search keyword.", required: false
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
