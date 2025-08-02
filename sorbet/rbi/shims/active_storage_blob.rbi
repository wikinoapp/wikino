# typed: strict
# frozen_string_literal: true

# This shim is for the BlobProcessable module that is dynamically included into ActiveStorage::Blob
class ActiveStorage::Blob
  include BlobProcessable
end