# frozen_string_literal: true

module Canary
  class NonotoSchema < GraphQL::Schema
    mutation Canary::Types::MutationType
    query Canary::Types::QueryType

    # Opt in to the new runtime (default in future graphql-ruby versions)
    use GraphQL::Execution::Interpreter
    use GraphQL::Analysis::AST

    # Add built-in connections for pagination
    use GraphQL::Pagination::Connections
  end
end
