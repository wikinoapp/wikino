# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :draft_page_record do
    space_record
    page_record
    space_member_record
    topic_record
    sequence(:title) { |n| "Page #{n}" }
    sequence(:body) { |n| "Body #{n}" }
    sequence(:body_html) { |n| "<div>Body #{n}</div>" }
    linked_page_ids { [] }
    modified_at { Time.current }
  end
end
