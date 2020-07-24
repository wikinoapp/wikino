# frozen_string_literal: true

module Types
  module Fields
    class Base < GraphQL::Schema::Field
      argument_class Types::Arguments::Base
    end
  end
end
