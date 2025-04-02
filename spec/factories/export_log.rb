# typed: false
# frozen_string_literal: true

FactoryBot.define do
  factory :export_log do
    space
    export
    message { "Export started." }
    logged_at { Time.current }
  end
end
