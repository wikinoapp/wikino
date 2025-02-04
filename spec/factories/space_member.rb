# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :space_member do
    space
    user
    role { SpaceMemberRole::Owner.serialize }
    joined_at { Time.current }

    trait :owner do
      role { SpaceMemberRole::Owner.serialize }
    end
  end
end
