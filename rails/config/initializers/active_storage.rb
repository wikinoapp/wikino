# typed: true
# frozen_string_literal: true

Rails.application.configure do
  # vipsを使用して画像処理を行う
  config.active_storage.variant_processor = :vips
end
