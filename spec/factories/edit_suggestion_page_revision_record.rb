# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :edit_suggestion_page_revision_record do
    association :space_record
    association :edit_suggestion_page_record
    association :editor_space_member_record, factory: :space_member_record
    title { "編集提案ページタイトル" }
    body { "編集提案ページ本文" }
    body_html { "<p>編集提案ページ本文</p>" }
  end
end
