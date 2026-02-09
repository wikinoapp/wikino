# typed: strict
# frozen_string_literal: true

module Exports
  class DestroyService < ApplicationService
    sig { params(export_record_id: Types::DatabaseId).void }
    def call(export_record_id:)
      export_record = ExportRecord.find(export_record_id)

      export_record.status_records.destroy_all

      export_record.destroy!

      nil
    end
  end
end
