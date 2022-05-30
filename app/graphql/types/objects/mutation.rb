# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class Mutation < Types::Objects::Base
      field :createNote, mutation: Mutations::CreateNote
      field :deleteNote, mutation: Mutations::DeleteNote
      field :updateNote, mutation: Mutations::UpdateNote
    end
  end
end
