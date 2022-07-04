# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.1.2"

gem "rails", "~> 7.0.0"

gem "activerecord-session_store"
gem "addressable"
gem "bootsnap", require: false
gem "by_star"
gem "commonmarker" # Using github-markup
gem "email_validator"
gem "github-markup"
gem "graphql"
gem "graphql-batch"
gem "jwt"
gem "pg"
gem "puma"
gem "puma_worker_killer"
gem "pundit"
gem "rack-cors"
gem "rack-mini-profiler"
gem "sorbet-runtime"

group :development, :test do
  gem "dotenv-rails"
  gem "factory_bot_rails"
  gem "pry-rails"
  gem "rspec-mocks"
  gem "rspec-rails"
  gem "rspec_junit_formatter" # Using on CircleCI
  gem "rubocop-graphql", require: false
  gem "rubocop-rails", require: false
  gem "rubocop-rspec", require: false
  gem "rubocop-sorbet", require: false
  gem "standard"
end

group :development do
  gem "active_record_query_trace"
  gem "bullet"
  gem "listen" # Using with `rails s` since Rails 5
  gem "sorbet"
  gem "tapioca", require: false
  gem "unparser", require: false # Used by rubocop-sorbet
end

group :test do
  # Use < 0.18 until the following issue will be resolved.
  # https://github.com/codeclimate/test-reporter/issues/418
  gem "simplecov", "< 0.22", require: false
end

group :production do
  gem "lograge"
end
