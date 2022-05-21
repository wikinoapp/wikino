# frozen_string_literal: true

class Internal::GraphqlController < ActionController::API
  include Internal::Authenticatable

  def execute
    variables = ensure_hash(params[:variables])
    context = {
      viewer: current_user,
    }
    result = NonotoSchema.execute(params[:query], variables:, context:)

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
