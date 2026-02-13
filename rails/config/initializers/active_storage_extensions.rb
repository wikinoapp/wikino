# typed: true
# frozen_string_literal: true

Rails.configuration.to_prepare do
  ActiveSupport.on_load(:active_storage_blob) do
    include BlobProcessable
  end
end
