# frozen_string_literal: true

class ApplicationRepository
  def initialize(graphql_client:)
    @graphql_client = graphql_client
  end

  private

  attr_reader :graphql_client

  def query_path
    file_name = "#{self.class.name.underscore.delete_suffix('_repository')}.graphql"

    Rails.root.join("app", "queries", file_name)
  end

  def query
    @query ||= File.read(query_path)
  end

  def execute(variables: {})
    graphql_client.execute(query, variables: variables)
  end
end
