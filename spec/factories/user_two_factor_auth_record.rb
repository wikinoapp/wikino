# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user_two_factor_auth_record do
    user_record { association :user_record }
    secret { ROTP::Base32.random }
    enabled { false }
    enabled_at { nil }
    recovery_codes { [] }

    trait :enabled do
      enabled { true }
      enabled_at { Time.current }
      recovery_codes { 10.times.map { SecureRandom.alphanumeric(8).downcase } }
    end
  end
end