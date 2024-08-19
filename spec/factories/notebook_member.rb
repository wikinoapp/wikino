# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :notebook_member do
    space
    notebook
    user
    role { NotebookMemberRole::Admin.serialize }
    joined_at { Time.current }

    trait :admin do
      role { NotebookMemberRole::Admin.serialize }
    end

    trait :member do
      role { NotebookMemberRole::Member.serialize }
    end
  end
end
