# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "2.7.1"

gem "rails", "~> 6.0.0"

gem "activerecord-session_store"
gem "addressable"
gem "commonmarker" # Using github-markup
gem "devise"
gem "dry-struct"
gem "email_validator"
gem "enumerize"
gem "github-markup"
gem "graphql", ">= 1.10.0.pre3" # https://github.com/rmosolgo/graphql-ruby/pull/2640
gem "graphql-batch"
gem "jb"
gem "nokogiri"
gem "pg"
gem "puma"
gem "puma_worker_killer"
gem "rack-mini-profiler"
gem "view_component"

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
  gem "spring-commands-rspec", require: false
  gem "spring"
end

group :test do
  gem "capybara"
  gem "factory_bot_rails"
  gem "selenium-webdriver"
  # Use < 0.18 until the following issue will be resolved.
  # https://github.com/codeclimate/test-reporter/issues/418
  gem "simplecov", "< 0.18", require: false
  gem "timecop"
  gem "webdrivers", require: !ENV["CI"] # Added to run spec with Chrome on local machine
end

group :production do
  gem "lograge"
end
