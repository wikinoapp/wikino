# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :page_editor do
    space
    page
    space_member
    last_page_modified_at { Time.zone.now }
  end
end
