# typed: strict
# frozen_string_literal: true

class GenerateExportFilesJob < ApplicationJob
  queue_as :default

  sig { params(export_id: T::Wikino::DatabaseId).void }
  def perform(export_id:)
    export = Export.find(export_id)

    GenerateExportFilesService.new.call(export:)

    nil
  end
end
