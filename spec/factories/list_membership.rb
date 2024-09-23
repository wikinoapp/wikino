# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :list_membership do
    space
    list
    member { association :user }
    role { ListMemberRole::Admin.serialize }
    joined_at { Time.current }

    trait :admin do
      role { ListMemberRole::Admin.serialize }
    end

    trait :member do
      role { ListMemberRole::Member.serialize }
    end
  end
end
