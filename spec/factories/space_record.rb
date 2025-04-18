# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :space_record do
    sequence(:identifier) { |n| "identifier#{n}" }
    sequence(:name) { |n| "Space #{n}" }
    plan { Plan::Small.serialize }
    joined_at { Time.current }

    trait :free do
      plan { Plan::Free.serialize }
    end

    trait :small do
      plan { Plan::Small.serialize }
    end

    trait :large do
      plan { Plan::Large.serialize }
    end
  end
end
