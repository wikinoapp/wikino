# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :link do
    note
    target_note factory: :note
  end
end
