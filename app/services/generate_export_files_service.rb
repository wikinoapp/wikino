# typed: strict
# frozen_string_literal: true

class GenerateExportFilesService < ApplicationService
  class Result < T::Struct
    const :export, Export
  end

  sig { params(export: Export, locale: String).returns(Result) }
  def call(export:, locale:)
    I18n.with_locale(locale) do
      if export.failed?
        export.add_log!(message: fetch_message(:already_finished_as_failed))
        return Result.new(export:)
      end

      if export.succeeded?
        export.add_log!(message: fetch_message(:already_finished_as_succeeded))
        return Result.new(export:)
      end

      ActiveRecord::Base.transaction do
        export.change_status!(kind: ExportStatusKind::Started)
        export.add_log!(message: fetch_message(:started))
      end

      target_pages = export.target_pages
      export.add_log!(message: fetch_message(:target_pages_count, count: target_pages.count))

      ActiveRecord::Base.transaction do
        export.change_status!(kind: ExportStatusKind::Succeeded)
        export.add_log!(message: fetch_message(:succeeded))
      end
    end
  rescue => exception
    ActiveRecord::Base.transaction do
      export.change_status!(kind: ExportStatusKind::Failed)
      export.add_log!(message: fetch_message(:failed))
    end

    Sentry.capture_exception(exception)
  ensure
    Result.new(export:)
  end

  sig { params(key: Symbol, args: T.untyped).returns(String) }
  private def fetch_message(key, **args)
    I18n.t("messages.export_logs.#{key}", **args)
  end
end
