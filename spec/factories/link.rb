# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :link do
    page
    target_page factory: :page
  end
end
