# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :list do
    space
    sequence(:identifier) { |n| "identifier_#{n}" }
    sequence(:name) { |n| "List #{n}" }
    description { "Note description" }
    visibility { ListVisibility::Public.serialize }

    trait :public do
      visibility { ListVisibility::Public.serialize }
    end

    trait :private do
      visibility { ListVisibility::Private.serialize }
    end
  end
end
