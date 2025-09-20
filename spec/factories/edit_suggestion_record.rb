# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :edit_suggestion_record do
    space_record
    topic_record
    association :created_user_record, factory: :user_record
    title { "編集提案タイトル" }
    description { "編集提案の説明" }
    status { EditSuggestionStatus::Draft.serialize }

    trait :open do
      status { EditSuggestionStatus::Open.serialize }
    end

    trait :applied do
      status { EditSuggestionStatus::Applied.serialize }
      applied_at { Time.current }
    end

    trait :closed do
      status { EditSuggestionStatus::Closed.serialize }
    end
  end
end
