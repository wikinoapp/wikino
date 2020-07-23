# frozen_string_literal: true

class ApplicationController < ActionController::Base
  private

  def graphql_client
    @graphql_client ||= Nonoto::Graphql::InternalClient.new(
      viewer: current_user
    )
  end
end
