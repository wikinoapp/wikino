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
      ActiveRecord::Base.transaction do
        export.change_status!(kind: ExportStatusKind::Started)
        export.add_log!(message: fetch_message(:started))
      end

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

        # 100件ごとにログを追加
        if (index + 1) % 100 == 0
          export.add_log!(message: fetch_message(:exported_page, progress: "#{index + 1}/#{total_count}"))
        end
      end

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

    if Rails.env.development?
      raise exception
    end

    Sentry.capture_exception(exception)
  end

  sig { params(key: Symbol, args: T.untyped).returns(String) }
  private def fetch_message(key, **args)
    I18n.t("messages.export_logs.#{key}", **args)
  end
end
