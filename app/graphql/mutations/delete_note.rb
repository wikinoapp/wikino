# typed: strict
# frozen_string_literal: true

module Mutations
  class DeleteNote < Mutations::Base
    extend T::Sig

    argument :id, ID, required: true

    field :errors, [Types::Objects::MutationErrorType], null: false

    sig { params(id: String).returns(T::Hash[Symbol, T.untyped]) }
    def resolve(id:)
      note = NonotoSchema.object_from_id(id)
      form = Forms::NoteDestruction.new(user: context[:viewer], note:)

      result = ActiveRecord::Base.transaction do
        Commands::DestroyNote.new(form:).run
      end

      {
        errors: result.errors.map { |error| { message: error.message } }
      }
    end
  end
end
