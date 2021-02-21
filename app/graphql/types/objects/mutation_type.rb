# frozen_string_literal: true

module Types
  module Objects
    class MutationType < Types::Objects::Base
      field :confirmEmail, mutation: Mutations::ConfirmEmail
      field :authenticate, mutation: Mutations::Authenticate

      field :createNote, mutation: Mutations::CreateNote
      field :deleteNote, mutation: Mutations::DeleteNote
      field :updateNote, mutation: Mutations::UpdateNote
    end
  end
end
