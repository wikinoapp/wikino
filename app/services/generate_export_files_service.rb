# typed: strict
# frozen_string_literal: true

class GenerateExportFilesService < ApplicationService
  sig { params(export: Export).void }
  def call(export:)
    if export.failed? || export.succeeded?
      return
    end

    export.change_status!(kind: ExportStatusKind::Started)

    output_path = output_to_files!(export:)
    zip_path = make_zip_file!(export:, output_path:)
    upload_zip_file!(export:, zip_path:)

    export.change_status!(kind: ExportStatusKind::Succeeded)
  rescue => exception
    export.change_status!(kind: ExportStatusKind::Failed)

    if Rails.env.development?
      raise exception
    end

    Sentry.capture_exception(exception)
  end

  sig { params(export: Export).returns(String) }
  private def output_to_files!(export:)
    target_pages = export.target_pages.preload(:topic)

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
    end

    export_base_dir.to_s
  end

  sig { params(export: Export, output_path: String).returns(String) }
  private def make_zip_file!(export:, output_path:)
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

    zip_file_path
  end

  sig { params(export: Export, zip_path: String).void }
  private def upload_zip_file!(export:, zip_path:)
    file = File.open(zip_path)

    export.file.attach(
      io: file,
      filename: File.basename(zip_path),
      content_type: "application/zip"
    )

    file.close
  end
end
