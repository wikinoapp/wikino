# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "2.7.2"

gem "rails", "~> 6.1.3"

gem "activerecord-session_store"
gem "addressable"
gem 'bootsnap', '>= 1.4.4', require: false
gem "by_star"
gem "commonmarker" # Using github-markup
gem "email_validator"
gem "github-markup"
gem "graphql", ">= 1.10.0.pre3" # https://github.com/rmosolgo/graphql-ruby/pull/2640
gem "graphql-batch"
gem "pg"
gem "puma"
gem "puma_worker_killer"
gem "rack-cors"
gem "rack-mini-profiler"

group :development, :test do
  gem "awesome_print"
  gem "dotenv-rails"
  gem "pry-rails"
  gem "rspec-mocks"
  gem "rspec-rails"
  gem "rspec_junit_formatter" # Using on CircleCI
end

group :development do
  gem "active_record_query_trace"
  gem "annotate"
  gem "better_errors"
  gem "binding_of_caller" # Using better_errors
  gem "bullet"
  gem "listen" # Using with `rails s` since Rails 5
  gem "rubocop"
  gem "solargraph"
  gem "spring"
  gem "spring-commands-rspec", require: false
end

group :test do
  gem "factory_bot_rails"
  # Use < 0.18 until the following issue will be resolved.
  # https://github.com/codeclimate/test-reporter/issues/418
  gem "simplecov", "< 0.18", require: false
end

group :production do
  gem "lograge"
end
