# typed: strict
# frozen_string_literal: true

class ExportService < ApplicationService
  class Result < T::Struct
    const :export, ExportRecord
  end

  sig do
    params(space: SpaceRecord, queued_by: SpaceMemberRecord).returns(Result)
  end
  def call(space:, queued_by:)
    export = ActiveRecord::Base.transaction do
      e = space.export_records.create!(
        queued_by_record:
      )
      e.change_status!(kind: ExportStatusKind::Queued)
      e
    end

    GenerateExportFilesJob.perform_later(export_id: export.id)

    Result.new(export:)
  end
end
