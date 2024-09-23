# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user do
    space
    sequence(:email) { |n| "test_#{n}@example.com" }
    sequence(:atname) { |n| "atname_#{n}" }
    role { UserRole::Owner.serialize }
    sequence(:name) { |n| "Name #{n}" }
    sequence(:description) { |n| "Description #{n}" }
    locale { UserLocale::Ja.serialize }
    time_zone { "Asia/Tokyo" }
    joined_at { Time.current }

    trait :owner do
      role { UserRole::Owner.serialize }
    end
  end
end
