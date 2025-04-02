# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :export do
    space
    started_by factory: :space_member
    started_at { Time.current }
  end
end
