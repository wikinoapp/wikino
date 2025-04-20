# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :page do
    space
    topic
    sequence(:number) { |n| n }
    sequence(:title) { |n| "Page #{n}" }
    sequence(:body) { |n| "Body #{n}" }
    sequence(:body_html) { |n| "<div>Body #{n}</div>" }
    linked_page_ids { [] }
    modified_at { Time.current }

    trait :published do
      published_at { modified_at }
    end

    trait :not_published do
      published_at { nil }
    end

    trait :trashed do
      trashed_at { Time.current }
    end
  end
end
