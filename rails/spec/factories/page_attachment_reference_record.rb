# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :page_attachment_reference_record do
    page_record
    attachment_record
  end
end
