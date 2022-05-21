# frozen_string_literal: true

FactoryBot.define do
  factory :note do
    user
    sequence(:title) { |n| "Note #{n}" }
    modified_at { Time.zone.now }
  end
end
