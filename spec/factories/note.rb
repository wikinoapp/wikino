# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :note do
    user
    sequence(:title) { |n| "Note #{n}" }
    modified_at { Time.zone.now }

    trait :with_content do
      after(:create) do |note|
        create(:note_content, user: note.user, note:)
      end
    end
  end
end
