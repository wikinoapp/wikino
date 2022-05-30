# typed: strict
# frozen_string_literal: true

module Types
  module Unions
    class DeleteNoteError < Types::Unions::Base
      extend T::Sig

      possible_types Types::Objects::MutationError

      sig { params(_object: DestroyNoteService::Error, _context: GraphQL::Query::Context).returns(T.class_of(Types::Objects::MutationError)) }
      def self.resolve_type(_object, _context)
        Types::Objects::MutationError
      end
    end
  end
end
