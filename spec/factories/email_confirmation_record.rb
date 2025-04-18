# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :email_confirmation_record do
    sequence(:email) { |n| "test_#{n}@example.com" }
    event { EmailConfirmationEvent::SignUp.serialize }
    code { "123456" }
    started_at { Time.zone.now }
  end
end
