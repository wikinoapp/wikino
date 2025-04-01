# typed: strict
# frozen_string_literal: true

class ExportService < ApplicationService
  class Result < T::Struct
    const :export, Export
  end

  sig do
    params(
      space: Space,
      started_by: SpaceMember,
      locale: ViewerLocale
    ).returns(Result)
  end
  def call(space:, started_by:, locale:)
    export = space.exports.create!(
      started_by:,
      started_at: Time.current
    )

    GenerateExportFilesJob.perform_later(
      export_id: export.id,
      locale: locale.serialize
    )

    Result.new(export:)
  end
end
