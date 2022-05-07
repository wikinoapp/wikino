# frozen_string_literal: true

Rails.application.routes.draw do
  use_doorkeeper
  constraints(format: "json") do
    constraints(subdomain: "api") do
      match "/internal/graphql", via: :post, as: :internal_graphql, to: "internal/graphql#execute"
    end

    if Rails.env.development?
      match "/local/graphql", via: :post, as: :local_graphql, to: "local/graphql#execute"
    end
  end
end
