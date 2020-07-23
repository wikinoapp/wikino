# frozen_string_literal: true

module Types
  module Object
    class QueryType < Types::Object::Base
      field :viewer, Types::Object::UserType, null: false

      def viewer
        context[:viewer]
      end
    end
  end
end
