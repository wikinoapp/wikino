# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :edit_suggestion_page_record, class: "EditSuggestionPageRecord" do
    association :space, factory: :space_record
    association :edit_suggestion, factory: :edit_suggestion_record
    association :page, factory: :page_record

    title_before { "変更前のタイトル" }
    title_after { "変更後のタイトル" }
    body_before { "変更前の本文" }
    body_after { "変更後の本文" }

    trait :new_page do
      page { nil }
      title_before { nil }
      body_before { nil }
      title_after { "新規ページタイトル" }
      body_after { "新規ページ本文" }
    end
  end
end
