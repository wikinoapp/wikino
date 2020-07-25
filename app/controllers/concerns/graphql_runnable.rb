# frozen_string_literal: true

module GraphqlRunnable
  def graphql_client
    @graphql_client ||= Nonoto::Graphql::InternalClient.new(
      viewer: current_user
    )
  end
end
