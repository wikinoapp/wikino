# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :notebook do
    space
    sequence(:identifier) { |n| "identifier_#{n}" }
    sequence(:name) { |n| "Notebook #{n}" }
    description { "Note description" }
    visibility { NotebookVisibility::Public.serialize }

    trait :public do
      visibility { NotebookVisibility::Public.serialize }
    end

    trait :private do
      visibility { NotebookVisibility::Private.serialize }
    end
  end
end
