# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :edit_suggestion_page_record do
    space_record
    edit_suggestion_record
    page_record
    page_revision_record

    trait :new_page do
      page_record { nil }
      page_revision_record { nil }
    end
  end
end
