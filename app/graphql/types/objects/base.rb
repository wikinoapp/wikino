# typed: true
# frozen_string_literal: true

module Types
  module Objects
    class Base < GraphQL::Schema::Object
      field_class Types::Fields::Base

      def database_id
        object.id
      end
    end
  end
end
