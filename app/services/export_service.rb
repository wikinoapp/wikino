# typed: strict
# frozen_string_literal: true

class ExportService < ApplicationService
  class Result < T::Struct
    const :export, Export
  end

  sig do
    params(
      space: Space,
      queued_by: SpaceMember,
      locale: ViewerLocale
    ).returns(Result)
  end
  def call(space:, queued_by:, locale:)
    export = ActiveRecord::Base.transaction do
      e = space.exports.create!(
        queued_by:
      )

      e.statuses.create!(
        kind: ExportStatusKind::Queued.serialize,
        changed_at: Time.current
      )

      e
    end

    GenerateExportFilesJob.perform_later(
      export_id: export.id,
      locale: locale.serialize
    )

    Result.new(export:)
  end
end
