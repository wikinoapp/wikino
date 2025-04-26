# typed: strict
# frozen_string_literal: true

class GenerateExportFilesJob < ApplicationJob
  queue_as :default

  sig { params(export_record_id: T::Wikino::DatabaseId).void }
  def perform(export_record_id:)
    export_record = ExportRecord.find(export_record_id)

    GenerateExportFilesService.new.call(export_record:)

    nil
  end
end
