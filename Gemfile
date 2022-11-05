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
gem "cssbundling-rails"
gem "github-markup"
gem "jsbundling-rails"
gem "omniauth-auth0"
gem "omniauth-rails_csrf_protection"
gem "pg"
gem "propshaft"
gem "puma"
gem "puma_worker_killer"
gem "pundit"
gem "rack-cors"
gem "rack-mini-profiler"
gem "sorbet-runtime"
gem "view_component"

group :development, :test do
  gem "dotenv-rails"
  gem "factory_bot_rails"
  gem "pry-rails"
  gem "rspec-mocks"
  gem "rspec-rails"
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

group :production do
  gem "lograge"
end
