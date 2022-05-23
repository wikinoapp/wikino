# frozen_string_literal: true
# typed: false

namespace :sorbet do
  task update: :environment do
    system "bundle exec srb rbi sorbet-typed"

    system "bundle exec tapioca gem"

    system "bundle exec rails rails_rbi:routes"
    system "bundle exec rails rails_rbi:models"
    system "bundle exec rails rails_rbi:mailers"
    system "bundle exec rails rails_rbi:jobs"
    system "bundle exec rails rails_rbi:custom"

    system "bundle exec tapioca todo"
    system "bundle exec srb rbi suggest-typed"
  end
end
