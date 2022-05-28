# typed: strict
# frozen_string_literal: true

module Mutations
  class CreateNote < Mutations::Base
    extend T::Sig

    argument :title, String, required: true
    argument :body, String, required: false

    field :note, Types::Objects::NoteType, null: true
    field :errors, [Types::Objects::MutationErrorType], null: false

    sig { params(title: String, body: T.nilable(String)).returns(T::Hash[Symbol, T.untyped]) }
    def resolve(title:, body: nil)
      user = context[:viewer]
      form = Forms::Note.new(user:, title:, body:)

      result = ActiveRecord::Base.transaction do
        Commands::CreateNote.new(user:, form:).run
      end

      {
        note: result.note,
        errors: result.errors.map { |error| { message: error.message } }
      }
    end
  end
end
