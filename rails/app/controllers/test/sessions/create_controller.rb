# typed: strict
# frozen_string_literal: true

module Test
  module Sessions
    class CreateController < ApplicationController
      sig { returns(T.untyped) }
      def call
        attrs = params.permit(session_attrs: {})

        attrs[:session_attrs].each do |key, value|
          session[key] = value
        end

        head :created
      end
    end
  end
end
