# typed: strict
# frozen_string_literal: true

class GenerateExportFilesService < ApplicationService
  class Result < T::Struct
    const :export, Export
  end

  sig { params(export: Export, locale: String).returns(Result) }
  def call(export:, locale:)
    I18n.with_locale(locale) do
      export.add_log!(message_key: :started, logged_at: Time.current)

      if export.finished?
        export.add_log!(message_key: :already_finished, logged_at: Time.current)
        return Result.new(export:)
      end

      ActiveRecord::Base.transaction do
        export.add_log!(message_key: :finished, logged_at: Time.current)
        export.update!(finished_at: Time.current)
      end

      Result.new(export:)
    end
  end
end
