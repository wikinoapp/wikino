# frozen_string_literal: true

FactoryBot.define do
  factory :user do
    sequence(:email) { |n| "user_#{n}@example.com" }
    encrypted_password { "password" }
    signed_up_at { Time.current }
  end
end
