# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :export do
    space
    queued_by factory: :space_member
  end
end
