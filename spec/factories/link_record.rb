# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :link_record do
    page_record
    target_page_record factory: :page_record
  end
end
