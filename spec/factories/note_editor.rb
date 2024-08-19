# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :note_editor do
    space
    note
    user
    last_note_modified_at { Time.current }
  end
end
