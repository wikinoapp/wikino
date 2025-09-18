# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :edit_suggestion_comment_record, class: "EditSuggestionCommentRecord" do
    association :space, factory: :space_record
    association :edit_suggestion, factory: :edit_suggestion_record
    association :created_user, factory: :user_record
    body { "コメントの本文" }
    body_html { "<p>コメントの本文</p>" }
  end
end
