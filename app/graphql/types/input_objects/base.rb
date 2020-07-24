# frozen_string_literal: true

module Types
  module InputObjects
    class Base < GraphQL::Schema::InputObject
      argument_class Types::Arguments::Base
    end
  end
end
