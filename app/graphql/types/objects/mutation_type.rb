# frozen_string_literal: true

module Types
  module Objects
    class MutationType < Types::Objects::Base
      field :createNote, mutation: Mutations::CreateNote
    end
  end
end
