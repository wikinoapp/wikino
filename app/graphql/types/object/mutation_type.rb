# frozen_string_literal: true

module Types
  module Object
    class MutationType < Types::Object::Base
      field :createNote, mutation: Mutations::CreateNote
    end
  end
end
