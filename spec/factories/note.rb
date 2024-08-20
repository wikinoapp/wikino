# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :note do
    space
    list
    author { association :user }
    sequence(:number) { |n| n }
    sequence(:title) { |n| "Note #{n}" }
    modified_at { Time.current }
  end
end
