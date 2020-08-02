# frozen_string_literal: true

module Types
  module Objects
    class MutationType < Types::Objects::Base
      field :createNote, mutation: Mutations::CreateNote
      field :updateNote, mutation: Mutations::UpdateNote
    end
  end
end
