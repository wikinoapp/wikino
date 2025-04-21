# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :topic_record do
    space_record
    sequence(:number) { |n| n }
    sequence(:name) { |n| "Topic #{n}" }
    description { "Page description" }
    visibility { TopicVisibility::Public.serialize }

    trait :public do
      visibility { TopicVisibility::Public.serialize }
    end

    trait :private do
      visibility { TopicVisibility::Private.serialize }
    end
  end
end
