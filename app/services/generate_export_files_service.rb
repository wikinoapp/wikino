# typed: strict
# frozen_string_literal: true

class GenerateExportFilesService < ApplicationService
  sig { params(export: Export, locale: String).void }
  def call(export:, locale:)
    I18n.with_locale(locale) do
      if export.failed?
        export.add_log!(message: fetch_message(:already_finished_as_failed))
        return
      end

      if export.succeeded?
        export.add_log!(message: fetch_message(:already_finished_as_succeeded))
        return
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
  end

  sig { params(key: Symbol, args: T.untyped).returns(String) }
  private def fetch_message(key, **args)
    I18n.t("messages.export_logs.#{key}", **args)
  end
end
