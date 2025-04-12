# typed: strict
# frozen_string_literal: true

class ExportService < ApplicationService
  class Result < T::Struct
    const :export, Export
  end

  sig do
    params(space: Space, queued_by: SpaceMember).returns(Result)
  end
  def call(space:, queued_by:)
    export = ActiveRecord::Base.transaction do
      e = space.exports.create!(
        queued_by:
      )
      e.change_status!(kind: ExportStatusKind::Queued)
      e
    end

    GenerateExportFilesJob.perform_later(export_id: export.id)

    Result.new(export:)
  end
end
