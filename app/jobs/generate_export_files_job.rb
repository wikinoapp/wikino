# typed: strict
# frozen_string_literal: true

class GenerateExportFilesJob < ApplicationJob
  queue_as :default

  sig { params(export_id: T::Wikino::DatabaseId, locale: String).void }
  def perform(export_id:, locale:)
    export = Export.find(export_id)

    GenerateExportFilesService.new.call(export:, locale:)

    nil
  end
end
