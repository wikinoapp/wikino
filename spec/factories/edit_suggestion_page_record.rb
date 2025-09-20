# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :edit_suggestion_page_record do
    space_record
    edit_suggestion_record
    page_record
    page_revision_record
    title { "変更後のタイトル" }
    body { "変更後の本文" }

    trait :new_page do
      page_record { nil }
      page_revision_record { nil }
      title { "新規ページタイトル" }
      body { "新規ページ本文" }
    end
  end
end
