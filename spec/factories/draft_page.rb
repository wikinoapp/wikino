# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :draft_page do
    space
    page
    editor { association :space_member }
    topic
    sequence(:title) { |n| "Page #{n}" }
    sequence(:body) { |n| "Body #{n}" }
    sequence(:body_html) { |n| "<div>Body #{n}</div>" }
    linked_page_ids { [] }
    modified_at { Time.current }
  end
end
