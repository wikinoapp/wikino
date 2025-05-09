# typed: strict
# frozen_string_literal: true

module PageService
  class BulkDestroyTrashed < ApplicationService
    sig { void }
    def call
      PageRecord.where(trashed_at: ..30.days.ago).destroy_all_with_related_records!
    end
  end
end
