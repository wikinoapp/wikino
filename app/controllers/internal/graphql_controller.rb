# typed: strict
# frozen_string_literal: true

module Internal
  class GraphqlController < ActionController::API
    extend T::Sig

    include Internal::Authenticatable

    sig { returns(T.untyped) }
    def execute
      variables = ensure_hash(params[:variables])
      context = {
        viewer: current_user
      }
      result = NonotoSchema.execute(params[:query].to_s, variables:, context:)

      render json: result
    end

    private

    sig do
      params(
        ambiguous_param: T.nilable(T.any(String, T::Hash[T.untyped, T.untyped], Numeric, ActionController::Parameters))
      )
        .returns(T.any(T::Hash[T.untyped, T.untyped], ActionController::Parameters))
    end
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
        raise(ArgumentError, "Unexpected parameter: #{ambiguous_param.inspect}")
      end
    end
  end
end
