# typed: strict
# frozen_string_literal: true

module SpaceService
  class Export < ApplicationService
    class Result < T::Struct
      const :export_record, ExportRecord
    end

    sig do
      params(space_record: SpaceRecord, queued_by_record: SpaceMemberRecord).returns(Result)
    end
    def call(space_record:, queued_by_record:)
      created_export_record = ActiveRecord::Base.transaction do
        export_record = space_record.export_records.create!(queued_by_record:)
        export_record.change_status!(kind: ExportStatusKind::Queued)
        export_record
      end

      GenerateExportFilesJob.perform_later(export_record_id: created_export_record.id)

      Result.new(export_record: created_export_record)
    end
  end
end
