# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :active_storage_attachment, class: "ActiveStorage::Attachment" do
    name { "file" }
    association :blob, factory: :active_storage_blob
    record { nil }
  end
end