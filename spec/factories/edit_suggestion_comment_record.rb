# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :edit_suggestion_comment_record do
    space_record
    edit_suggestion_record
    association :created_user_record, factory: :user_record
    body { "コメントの本文" }
    body_html { "<p>コメントの本文</p>" }
  end
end
