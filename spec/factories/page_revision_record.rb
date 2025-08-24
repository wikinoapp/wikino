# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :page_revision_record do
    space_record
    page_record
    space_member_record
    sequence(:body) { |n| "Body #{n}" }
    body_html { "<p>#{body}</p>" }
  end
end
