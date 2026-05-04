# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :topic_member_record do
    space_record
    topic_record
    space_member_record
    joined_at { Time.current }
    scopes { [] }
  end
end
