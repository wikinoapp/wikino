# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :user_password do
    user
    password { "passw0rd" }
  end
end
