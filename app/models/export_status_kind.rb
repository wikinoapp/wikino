# typed: strict
# frozen_string_literal: true

class ExportStatusKind < T::Enum
  extend T::Sig

  enums do
    Queued = new("queued")
    Started = new("started")
    Succeeded = new("succeeded")
    Failed = new("failed")
  end
end
