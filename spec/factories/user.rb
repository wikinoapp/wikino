# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user do
    sequence(:auth0_user_id) { |n| "auth0|00000000000000000000000#{n}" }
  end
end
