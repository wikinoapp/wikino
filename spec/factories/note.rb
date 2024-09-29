# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :note do
    space
    topic
    sequence(:number) { |n| n }
    sequence(:title) { |n| "Note #{n}" }
    sequence(:body) { |n| "Body #{n}" }
    sequence(:body_html) { |n| "<div>Body #{n}</div>" }
    linked_note_ids { [] }
    modified_at { Time.current }
  end
end
