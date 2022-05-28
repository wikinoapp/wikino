# typed: false
# frozen_string_literal: true

namespace :sorbet do
  task update: :environment do
    system "bundle exec srb rbi sorbet-typed"

    system "bundle exec tapioca gem"
    system "bundle exec tapioca dsl"

    system "bundle exec tapioca todo"
    system "bundle exec srb rbi suggest-typed"
  end
end
