# typed: strict
# frozen_string_literal: true

class AttachmentProcessingStatus < T::Enum
  extend T::Sig

  enums do
    Pending = new("pending")
    Processing = new("processing")
    Completed = new("completed")
    Failed = new("failed")
  end
end
