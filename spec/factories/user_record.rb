# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user_record do
    sequence(:email) { |n| "test_#{n}@example.com" }
    sequence(:atname) { |n| "atname_#{n}" }
    sequence(:name) { |n| "Name #{n}" }
    sequence(:description) { |n| "Description #{n}" }
    locale { ViewerLocale::Ja.serialize }
    time_zone { "Asia/Tokyo" }
    joined_at { Time.current }

    trait :with_password do
      after(:create) do |user|
        create(:user_password_record, user_record: user)
      end
    end
  end
end
