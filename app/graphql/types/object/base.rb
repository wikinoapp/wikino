# frozen_string_literal: true

module Types
  module Object
    class Base < GraphQL::Schema::Object
      field_class Types::Field::Base

      def database_id
        object.id
      end
    end
  end
end
