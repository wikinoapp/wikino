# typed: strict
# frozen_string_literal: true

module Types
  module Unions
    class CreateNoteError < Types::Unions::Base
      extend T::Sig

      description "A mutation error."

      possible_types Types::Objects::MutationError, Types::Objects::DuplicatedNoteError

      sig do
        params(
          object: T.any(NoteUpsertable::Error, NoteUpsertable::DuplicatedNoteError),
          _context: GraphQL::Query::Context
        ).returns(T.any(T.class_of(Types::Objects::MutationError), T.class_of(Types::Objects::DuplicatedNoteError)))
      end
      def self.resolve_type(object, _context)
        case object
        when NoteUpsertable::DuplicatedNoteError
          Types::Objects::DuplicatedNoteError
        else
          Types::Objects::MutationError
        end
      end
    end
  end
end
