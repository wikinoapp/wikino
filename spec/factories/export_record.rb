# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :export_record do
    space_record
    queued_by_record factory: :space_member_record
  end
end
