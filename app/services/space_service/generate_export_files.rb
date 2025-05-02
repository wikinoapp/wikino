# typed: strict
# frozen_string_literal: true

require "zip"

module SpaceService
  class GenerateExportFiles < ApplicationService
    sig { params(export_record: ExportRecord).void }
    def call(export_record:)
      if export_record.failed? || export_record.succeeded?
        return
      end

      begin
        export_record.change_status!(kind: ExportStatusKind::Started)

        output_path = output_to_files!(export_record:)
        zip_path = make_zip_file!(export_record:, output_path:)
        upload_zip_file!(export_record:, zip_path:)

        export_record.change_status!(kind: ExportStatusKind::Succeeded)
      rescue => exception
        export_record.change_status!(kind: ExportStatusKind::Failed)
        raise exception
      end

      export_record.send_succeeded_mail!
    end

    sig { params(export_record: ExportRecord).returns(String) }
    private def output_to_files!(export_record:)
      target_pages = export_record.target_pages.preload(:topic_record)

      export_base_dir = Rails.root.join("tmp", "export_#{export_record.id}")
      FileUtils.mkdir_p(export_base_dir)

      topics = {}

      target_pages.find_each.with_index do |page, index|
        topic = page.topic_record.not_nil!

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

    sig { params(export_record: ExportRecord, output_path: String).returns(String) }
    private def make_zip_file!(export_record:, output_path:)
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

    sig { params(export_record: ExportRecord, zip_path: String).void }
    private def upload_zip_file!(export_record:, zip_path:)
      file = File.open(zip_path)

      export_record.file.attach(
        io: file,
        filename: File.basename(zip_path),
        content_type: "application/zip"
      )

      file.close
    end
  end
end
