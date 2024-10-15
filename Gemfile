# frozen_string_literal: true

source "https://rubygems.org"
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

ruby "3.3.4"

gem "rails", "~> 7.2.1"

gem "activerecord-session_store"
gem "activerecord_cursor_paginate"
gem "addressable"
gem "bcrypt" # `has_secure_password` で使っている
gem "bootsnap", require: false
gem "by_star"
gem "commonmarker"
gem "cssbundling-rails"
gem "discard"
gem "email_validator"
gem "http_accept_language"
gem "inline_svg"
gem "jsbundling-rails"
gem "meta-tags"
gem "pg"
gem "propshaft"
gem "puma"
gem "pundit"
gem "rack-cors"
gem "rails-i18n"
gem "sequenced"
gem "sorbet-runtime"
gem "sorbet-struct-comparable"
gem "strong_migrations"
gem "view_component"

group :development, :test do
  gem "dotenv-rails"
  gem "factory_bot_rails"
  gem "rspec-rails"
  gem "rubocop-factory_bot", require: false
  gem "rubocop-rails", require: false
  gem "rubocop-rspec", require: false
  gem "standard"
  gem "standard-rails"
  gem "standard-sorbet"
end

group :development do
  gem "bullet"
  gem "letter_opener_web"
  gem "sorbet"
  gem "tapioca", require: false
end

group :test do
  gem "cuprite"
  gem "capybara"
  gem "vcr"
end

group :production do
  gem "lograge"
  gem "resend"
end
