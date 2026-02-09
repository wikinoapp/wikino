# typed: strict
# frozen_string_literal: true

require "zip"

module Spaces
  class GenerateExportFilesService < ApplicationService
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

      send_succeeded_mail!(export_record:)
    end

    sig { params(export_record: ExportRecord).returns(String) }
    private def output_to_files!(export_record:)
      target_pages = export_record.target_pages.preload(:topic_record, page_attachment_reference_records: {attachment_record: :active_storage_attachment_record})

      export_base_dir = Rails.root.join("tmp", "export_#{export_record.id}")
      FileUtils.mkdir_p(export_base_dir)

      topics = {}
      # 各トピックごとの添付ファイルIDとファイル名のマッピングを保持
      topic_attachment_mappings = T.let({}, T::Hash[String, T::Hash[String, String]])

      # トピックごとにページと添付ファイルを整理
      pages_by_topic = target_pages.group_by(&:topic_record)

      pages_by_topic.each do |topic, pages|
        # トピックディレクトリを作成
        topic = topic.not_nil!
        topic_dir = File.join(export_base_dir, topic.name)
        FileUtils.mkdir_p(topic_dir)
        topics[topic.id] = topic_dir

        # トピック内の添付ファイルディレクトリを作成
        attachments_dir = File.join(topic_dir, "attachments")
        FileUtils.mkdir_p(attachments_dir)

        # このトピックのページから参照されている添付ファイルを収集
        topic_attachment_records = T.let([], T::Array[AttachmentRecord])
        attachment_id_to_filename = T.let({}, T::Hash[String, String])

        pages.each do |page|
          page.page_attachment_reference_records.each do |reference|
            attachment_record = reference.attachment_record
            if attachment_record && !topic_attachment_records.include?(attachment_record)
              topic_attachment_records << attachment_record
            end
          end
        end

        # 添付ファイルをダウンロードして保存
        topic_attachment_records.each do |attachment_record|
          blob = attachment_record.blob_record
          next unless blob

          begin
            # ファイル名を取得（重複する場合は番号を付ける）
            original_filename = attachment_record.filename || "attachment"
            filename = original_filename
            counter = 1

            while File.exist?(File.join(attachments_dir, filename))
              # ファイル名に番号を追加（例: image.png → image_2.png）
              ext = File.extname(original_filename)
              basename = File.basename(original_filename, ext)
              filename = "#{basename}_#{counter}#{ext}"
              counter += 1
            end

            # ファイルをダウンロードして保存
            File.open(File.join(attachments_dir, filename), "wb") do |file|
              blob.download { |chunk| file.write(chunk) }
            end

            # IDとファイル名のマッピングを保存
            attachment_id_to_filename[attachment_record.id] = filename
          rescue => e
            Rails.logger.error("Failed to download attachment #{attachment_record.id}: #{e.message}")
          end
        end

        # このトピックのマッピングを保存
        topic_attachment_mappings[topic.id] = attachment_id_to_filename

        # ページのMarkdownファイルを作成
        pages.each do |page|
          # Markdown内のURLを相対パスに変換
          body_with_relative_paths = convert_attachment_urls(page.body, attachment_id_to_filename)

          filename = "#{page.title}.md"
          file_path = File.join(topic_dir, filename)

          File.write(file_path, body_with_relative_paths)
        end
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

    sig { params(export_record: ExportRecord).void }
    private def send_succeeded_mail!(export_record:)
      ExportMailer.succeeded(
        export_id: export_record.id,
        locale: export_record.queued_by_record.not_nil!.user_record_locale
      ).deliver_later
    end

    # Markdown内の添付ファイルURLを相対パスに変換
    sig { params(body: String, attachment_id_to_filename: T::Hash[String, String]).returns(String) }
    private def convert_attachment_urls(body, attachment_id_to_filename)
      converted_body = body.dup

      attachment_id_to_filename.each do |attachment_id, filename|
        # imgタグのsrc属性を変換
        # <img src="/attachments/id"> → <img src="attachments/filename">
        converted_body.gsub!(%r{(<img[^>]+src=["'])/attachments/#{Regexp.escape(attachment_id)}(["'][^>]*>)}, "\\1attachments/#{filename}\\2")

        # imgタグのwidth/height属性も考慮
        # <img width="600" height="400" alt="alt" src="/attachments/id"> → <img width="600" height="400" alt="alt" src="attachments/filename">
        converted_body.gsub!(%r{(<img[^>]+src=["'])/attachments/#{Regexp.escape(attachment_id)}(["'])}, "\\1attachments/#{filename}\\2")

        # aタグのhref属性を変換
        # <a href="/attachments/id"> → <a href="attachments/filename">
        converted_body.gsub!(%r{(<a[^>]+href=["'])/attachments/#{Regexp.escape(attachment_id)}(["'][^>]*>)}, "\\1attachments/#{filename}\\2")

        # Markdown形式の画像を変換
        # ![alt](/attachments/id) → ![alt](attachments/filename)
        converted_body.gsub!(%r{(!\[[^\]]*\]\()/attachments/#{Regexp.escape(attachment_id)}(\))}, "\\1attachments/#{filename}\\2")

        # Markdown形式のリンクを変換
        # [text](/attachments/id) → [text](attachments/filename)
        converted_body.gsub!(%r{((?<!!)\[[^\]]+\]\()/attachments/#{Regexp.escape(attachment_id)}(\))}, "\\1attachments/#{filename}\\2")
      end

      converted_body
    end
  end
end
