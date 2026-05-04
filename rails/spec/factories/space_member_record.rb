# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :space_member_record do
    space_record
    user_record
    active { true }
    joined_at { Time.current }
    scopes { ["space:admin"] }
  end
end
