# frozen_string_literal: true

FactoryBot.define do
  factory :note do
    user
    sequence(:title) { |n| "Note #{n}" }
    sequence(:body) { |n| "This is Note #{n}." }
    sequence(:body_html) { |n| "This is Note #{n}." }
  end
end
