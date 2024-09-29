# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :topic_membership do
    space
    topic
    member { association :user }
    role { TopicMemberRole::Admin.serialize }
    joined_at { Time.current }

    trait :admin do
      role { TopicMemberRole::Admin.serialize }
    end

    trait :member do
      role { TopicMemberRole::Member.serialize }
    end
  end
end
