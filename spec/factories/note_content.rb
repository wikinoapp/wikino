# frozen_string_literal: true

FactoryBot.define do
  factory :note_content do
    user
    note
    sequence(:body) { |n| "This is Note #{n}." }
    sequence(:body_html) { |n| "This is Note #{n}." }
  end
end
