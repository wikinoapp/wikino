# frozen_string_literal: true

module Api
  module Internal
    class GraphqlController < Api::Internal::ApplicationController
      before_action :authenticate_with_access_token
      skip_before_action :verify_authenticity_token

      def execute
        variables = ensure_hash(params[:variables])
        query = params[:query]
        context = {
          viewer: current_user
        }
        result = NonotoSchema.execute(query, variables: variables, context: context)

        render json: result
      end

      private

      def ensure_hash(ambiguous_param)
        case ambiguous_param
        when String
          if ambiguous_param.present?
            ensure_hash(JSON.parse(ambiguous_param))
          else
            {}
          end
        when Hash, ActionController::Parameters
          ambiguous_param
        when nil
          {}
        else
          raise ArgumentError, "Unexpected parameter: #{ambiguous_param}"
        end
      end
    end
  end
end
