# typed: strict
# frozen_string_literal: true

class GenerateExportFilesService < ApplicationService
  class Result < T::Struct
    const :export, Export
  end

  sig { params(export: Export, locale: String).returns(Result) }
  def call(export:, locale:)
    I18n.with_locale(locale) do
      export.add_log!(message: fetch_message(:started))

      if export.finished?
        export.add_log!(message: fetch_message(:already_finished))
        return Result.new(export:)
      end

      target_pages = export.target_pages
      export.add_log!(message: fetch_message(:target_pages_count, count: target_pages.count))

      ActiveRecord::Base.transaction do
        export.add_log!(message: fetch_message(:finished))
        export.update!(finished_at: Time.current)
      end

      Result.new(export:)
    end
  end

  sig { params(key: Symbol, args: T.untyped).returns(String) }
  private def fetch_message(key, **args)
    I18n.t("messages.export_logs.#{key}", **args)
  end
end
