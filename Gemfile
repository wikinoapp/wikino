# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "2.6.5"

gem "rails", "~> 6.0.0"

gem "activerecord-session_store"
gem "batch-loader"
gem "bootsnap", require: false
gem "commonmarker" # Using github-markup
gem "github-markup"
gem "graphql"
gem "meta-tags"
gem "pg"
gem "puma"
gem "puma_worker_killer"
gem "pundit"
gem "rails-i18n"

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
  gem "derailed_benchmarks"
  gem "graphiql-rails"
  gem "graphql-docs"
  gem "i18n-tasks"
  gem "listen" # Added by `rails s` since Rails 5
  gem "memory_profiler"
  gem "meta_request"
  gem "rubocop"
  gem "spring"
  gem "spring-commands-rspec", require: false
  gem "squasher"
  gem "stackprof"
  gem "traceroute"
end

group :test do
  gem "capybara"
  gem "factory_bot_rails"
  gem "selenium-webdriver"
  gem "simplecov", require: false
  gem "timecop"
  gem "webdrivers", require: !ENV["CI"] # Added to run spec with Chrome on local machine
end

group :production do
  gem "lograge"
end
