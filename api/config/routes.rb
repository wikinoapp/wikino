# frozen_string_literal: true

Rails.application.routes.draw do
  if Rails.env.development?
    scope module: :local_api do
      constraints(format: "json") do
        match "/api/local/graphql", via: :post, as: :graphql, to: "graphql#execute"
      end
    end
  end
end
