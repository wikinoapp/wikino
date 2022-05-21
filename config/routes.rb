# frozen_string_literal: true

Rails.application.routes.draw do
  constraints(subdomain: "api") do
    match "/internal/graphql", via: :post, as: :internal_graphql, to: "internal/graphql#execute"
  end
end
