# frozen_string_literal: true

class NonotoSchema < GraphQL::Schema
  mutation Types::Object::MutationType
  query Types::Object::QueryType

  use GraphQL::Execution::Interpreter
  use GraphQL::Analysis::AST
  use GraphQL::Pagination::Connections
end
