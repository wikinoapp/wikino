# frozen_string_literal: true

Rails.application.config.middleware.insert_before 0, Rack::Cors do
  allow do
    origins '*'
    resource '/internal/graphql',
      headers: :any,
      methods: [:post]
  end
end
