# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.4.4"

gem "rails", "~> 8.0.0"

gem "activerecord_cursor_paginate"
gem "addressable"
gem "aws-sdk-s3", require: false # Active Storageで使っている
gem "bcrypt" # `has_secure_password` で使っている
gem "bootsnap", require: false
gem "by_star"
gem "commonmarker" # `html-pipeline` で使っている
gem "cssbundling-rails"
gem "discard"
gem "email_validator"
gem "html-pipeline"
gem "http_accept_language"
gem "inline_svg"
gem "jsbundling-rails"
gem "meta-tags"
gem "mission_control-jobs"
gem "pg"
gem "propshaft"
gem "puma"
gem "rack-cors"
gem "rails-i18n"
gem "rotp"
gem "rqrcode"
gem "rubyzip"
gem "sentry-rails"
gem "sentry-ruby"
gem "sequenced"
gem "solid_queue"
gem "sorbet-runtime"
gem "sorbet-struct-comparable"
gem "stackprof" # Sentryで使っている
gem "strong_migrations"
gem "view_component"

group :development, :test do
  gem "dotenv-rails"
  gem "erb_lint", require: false
  gem "factory_bot_rails"
  gem "rspec-rails"
  gem "rubocop-factory_bot", require: false
  gem "rubocop-rails", require: false
  # Standard gemの `extend_config` が動くようになるまでバージョン固定する
  # https://github.com/standardrb/standard/issues/701
  gem "rubocop-rspec", "~> 3.6.0", require: false
  gem "standard"
  gem "standard-rails"
  gem "standard-sorbet"
end

group :development do
  gem "bullet"
  gem "letter_opener_web"
  gem "rails_live_reload"
  gem "ruby-lsp", require: false
  gem "sorbet"
  gem "tapioca", require: false
end

group :test do
  gem "capybara"
  gem "selenium-webdriver"
  gem "vcr"
end

group :production do
  gem "lograge"
  gem "resend"
end
