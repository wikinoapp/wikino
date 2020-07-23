# frozen_string_literal: true

module Nonoto
  module Graphql
    class InternalClient
      def initialize(viewer:)
        @viewer = viewer
      end

      def execute(query, variables: {})
        NonotoSchema.execute(query, variables: variables, context: context)
      end

      private

      attr_reader :viewer

      def context
        {
          viewer: viewer
        }
      end
    end
  end
end
