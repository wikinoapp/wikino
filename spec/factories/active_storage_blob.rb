# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :active_storage_blob, class: "ActiveStorage::Blob" do
    key { SecureRandom.uuid }
    filename { "test_file.txt" }
    content_type { "text/plain" }
    byte_size { 1024 }
    checksum { Digest::MD5.base64digest("test content") }
    created_at { Time.current }
  end
end