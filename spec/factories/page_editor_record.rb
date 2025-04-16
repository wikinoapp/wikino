# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :page_editor_record do
    space_record
    page_record
    space_member_record
    last_page_modified_at { Time.zone.now }
  end
end
