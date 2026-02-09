# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user_password_record do
    user_record
    password { "passw0rd" }
  end
end
