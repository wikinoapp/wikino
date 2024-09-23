# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :note_editorship do
    space
    note
    editor { association :user }
    last_note_modified_at { Time.zone.now }
  end
end
