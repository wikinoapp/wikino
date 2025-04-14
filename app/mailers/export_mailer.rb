# typed: true
# frozen_string_literal: true

class ExportMailer < ApplicationMailer
  sig { params(export_id: T::Wikino::DatabaseId, locale: String).void }
  def succeeded(export_id:, locale:)
    @export = ExportRecord.find(export_id)
    @space = @export.space

    I18n.with_locale(locale) do
      mail(
        to: @export.queued_by_record.not_nil!.user.not_nil!.email,
        subject: default_i18n_subject
      )
    end
  end
end
