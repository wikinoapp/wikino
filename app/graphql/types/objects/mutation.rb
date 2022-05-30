# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class Mutation < Types::Objects::Base
      field :create_note, mutation: Mutations::CreateNote
      field :delete_note, mutation: Mutations::DeleteNote
      field :update_note, mutation: Mutations::UpdateNote
    end
  end
end
