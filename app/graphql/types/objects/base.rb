# typed: strict
# frozen_string_literal: true

module Types
  module Objects
    class Base < GraphQL::Schema::Object
      extend T::Sig

      field_class Types::Fields::Base

      sig { returns(String) }
      def database_id
        object.id
      end
    end
  end
end
