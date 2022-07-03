# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class Mutation < Types::Objects::Base
      description "Modifying data operations."

      field :create_note, "Creates a note.", mutation: Mutations::CreateNote
      field :delete_note, "Deletes a note.", mutation: Mutations::DeleteNote
      field :update_note, "Updates a note.", mutation: Mutations::UpdateNote
    end
  end
end
