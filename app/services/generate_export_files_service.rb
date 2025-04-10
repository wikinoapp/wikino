# typed: strict
# frozen_string_literal: true

class GenerateExportFilesService < ApplicationService
  sig { params(export: Export, locale: String).void }
  def call(export:, locale:)
    I18n.with_locale(locale) do
      # すでに失敗していたらその旨をログに残して終了する
      if export.failed?
        export.add_log!(message: fetch_message(:already_finished_as_failed))
        return
      end

      # すでに成功していたらその旨をログに残して終了する
      if export.succeeded?
        export.add_log!(message: fetch_message(:already_finished_as_succeeded))
        return
      end

      # エクスポートを開始する
      chenge_status!(export:, kind: ExportStatusKind::Started)

      output_path = output_to_files!(export:)
      zip_path = make_zip_file!(export:, output_path:)
      upload_zip_file!(export:, zip_path:)

      chenge_status!(export:, kind: ExportStatusKind::Succeeded)
    end
  rescue => exception
    chenge_status!(export:, kind: ExportStatusKind::Failed)

    if Rails.env.development?
      raise exception
    end

    Sentry.capture_exception(exception)
  end

  sig { params(export: Export, kind: ExportStatusKind).void }
  private def chenge_status!(export:, kind:)
    ActiveRecord::Base.transaction do
      export.change_status!(kind:)
      export.add_log!(message: fetch_message(kind.serialize.to_sym))
    end
  end

  sig { params(key: Symbol, args: T.untyped).returns(String) }
  private def fetch_message(key, **args)
    I18n.t("messages.export_logs.#{key}", **args)
  end

  sig { params(export: Export).returns(String) }
  private def output_to_files!(export:)
    target_pages = export.target_pages.preload(:topic)
    total_count = target_pages.count
    export.add_log!(message: fetch_message(:target_pages_count, total_count:))

    export_base_dir = Rails.root.join("tmp", "export_#{export.id}")
    FileUtils.mkdir_p(export_base_dir)

    topics = {}

    target_pages.find_each.with_index do |page, index|
      topic = page.topic

      unless topics[topic.id]
        topic_dir = File.join(export_base_dir, topic.name)
        FileUtils.mkdir_p(topic_dir)
        topics[topic.id] = topic_dir
      end

      filename = "#{page.title}.md"
      file_path = File.join(topics[topic.id], filename)

      File.write(file_path, page.body)

      # 100件ごとか、すべてのページを変換したらログを追加
      if (index + 1) % 100 == 0 || (index + 1) == total_count
        export.add_log!(message: fetch_message(:output_to_file, progress: "#{index + 1}/#{total_count}"))
      end
    end

    export_base_dir.to_s
  end

  sig { params(export: Export, output_path: String).returns(String) }
  private def make_zip_file!(export:, output_path:)
    export.add_log!(message: fetch_message(:zipping_files))

    zip_file_path = "#{output_path}.zip"

    if File.exist?(zip_file_path)
      File.delete(zip_file_path)
    end

    Zip::File.open(zip_file_path, Zip::File::CREATE) do |zipfile|
      Dir[File.join(output_path, "**", "**")].each do |file|
        zipfile.add(file.sub("#{output_path}/", ""), file)
      end
    end

    FileUtils.rm_rf(output_path)

    export.add_log!(message: fetch_message(:zipped_files))

    zip_file_path
  end

  sig { params(export: Export, zip_path: String).void }
  private def upload_zip_file!(export:, zip_path:)
    export.add_log!(message: fetch_message(:uploading_zip_file))

    file = File.open(zip_path)
    export.file.attach(
      io: file,
      filename: File.basename(zip_path),
      content_type: "application/zip"
    )
    file.close

    export.add_log!(message: fetch_message(:uploaded_zip_file))
  end
end
