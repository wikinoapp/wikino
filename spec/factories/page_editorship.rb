# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :page_editorship do
    space
    page
    editor { association :user }
    last_page_modified_at { Time.zone.now }
  end
end
