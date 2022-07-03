# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class Mutation < Types::Objects::Base
      description "Modifying data operations."

      field :create_note, mutation: Mutations::CreateNote, description: "Creates a note."
      field :delete_note, mutation: Mutations::DeleteNote, description: "Deletes a note."
      field :update_note, mutation: Mutations::UpdateNote, description: "Updates a note."
    end
  end
end
