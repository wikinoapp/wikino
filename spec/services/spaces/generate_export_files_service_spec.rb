# typed: false
# frozen_string_literal: true

RSpec.describe Spaces::GenerateExportFilesService, type: :service do
  describe "#call" do
    it "エクスポートが失敗しているとき、何もしないこと" do
      export_record = create(:export_record, :failed)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Failed)

      Spaces::GenerateExportFilesService.new.call(export_record:)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Failed)
    end

    it "エクスポートが成功しているとき、何もしないこと" do
      export_record = create(:export_record, :succeeded)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Succeeded)

      Spaces::GenerateExportFilesService.new.call(export_record:)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Succeeded)
    end

    it "エクスポートが開始されていないとき、エクスポートを開始すること" do
      export_record = create(:export_record, :queued)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.count).to eq(1)
      expect(export_record.latest_status_kind).to eq(ExportStatusKind::Queued)

      Spaces::GenerateExportFilesService.new.call(export_record:)

      expect(ExportRecord.count).to eq(1)
      expect(export_record.status_records.order(changed_at: :asc).pluck(:kind)).to eq([
        ExportStatusKind::Queued.serialize,
        ExportStatusKind::Started.serialize,
        ExportStatusKind::Succeeded.serialize
      ])
    end

    context "添付ファイルを含むページのエクスポート" do
      it "添付ファイルをattachmentsディレクトリに含め、URLを相対パスに変換すること" do
        space_record = create(:space_record)
        space_member_record = create(:space_member_record, space_record:)
        topic_record = create(:topic_record, space_record:)
        page_record = create(:page_record, :published, space_record:, topic_record:)

        # 添付ファイルを作成
        blob = ActiveStorage::Blob.create_and_upload!(
          io: StringIO.new("test image content"),
          filename: "test-image.png",
          content_type: "image/png"
        )
        active_storage_attachment = ActiveStorage::Attachment.create!(
          name: "file",
          record: space_record,  # recordパラメータを使用
          blob:
        )

        attachment_record = create(
          :attachment_record,
          space_record:,
          attached_space_member_record: space_member_record,
          active_storage_attachment_id: active_storage_attachment.id
        )

        # ページ本文に添付ファイルへのリンクを含める
        page_body = <<~MARKDOWN
          # Test Page
          
          Here is an image:
          <img width="600" height="400" alt="test-image.png" src="/attachments/#{attachment_record.id}">
          
          And a link:
          [Download test-image.png](/attachments/#{attachment_record.id})
        MARKDOWN

        page_record.update!(body: page_body)

        # 添付ファイル参照を作成
        PageAttachmentReferenceRecord.create!(
          page_id: page_record.id,
          attachment_id: attachment_record.id
        )

        # エクスポートレコードを作成
        export_record = create(:export_record, :queued,
          space_record:,
          queued_by_record: space_member_record)

        # ZIPファイルをテスト用に一時ディレクトリに作成
        Dir.mktmpdir do |temp_dir|
          export_dir = File.join(temp_dir, "export_#{export_record.id}")
          allow(Rails.root).to receive(:join).and_call_original
          allow(Rails.root).to receive(:join).with("tmp", "export_#{export_record.id}").and_return(Pathname.new(export_dir))

          # サービスを実行
          Spaces::GenerateExportFilesService.new.call(export_record:)

          # ZIPファイルが作成されていることを確認
          zip_path = "#{export_dir}.zip"
          expect(File.exist?(zip_path)).to be true

          # ZIPファイルの内容を確認
          Zip::File.open(zip_path) do |zipfile|
            # attachmentsディレクトリが存在することを確認
            attachments_entry = zipfile.find_entry("attachments/test-image.png")
            expect(attachments_entry).not_to be_nil

            # ページファイルが存在することを確認
            page_entry = zipfile.find_entry("#{topic_record.name}/#{page_record.title}.md")
            expect(page_entry).not_to be_nil

            # ページ内容を確認
            page_content = zipfile.read(page_entry)

            # URLが相対パスに変換されていることを確認
            expect(page_content).to include('src="attachments/test-image.png"')
            expect(page_content).to include("[Download test-image.png](attachments/test-image.png)")

            # 絶対パスが残っていないことを確認
            expect(page_content).not_to include("/attachments/#{attachment_record.id}")
          end
        end
      end

      it "同じファイル名の添付ファイルがある場合、番号を付けて区別すること" do
        space_record = create(:space_record)
        space_member_record = create(:space_member_record, space_record:)
        topic_record = create(:topic_record, space_record:)
        page_record = create(:page_record, :published, space_record:, topic_record:)

        # 同じファイル名で2つの添付ファイルを作成
        # 1つ目の添付ファイル
        blob1 = ActiveStorage::Blob.create_and_upload!(
          io: StringIO.new("first image"),
          filename: "image.png",
          content_type: "image/png"
        )
        attachment1 = ActiveStorage::Attachment.create!(
          name: "file",
          record: space_record,
          blob: blob1
        )

        attachment_record1 = create(
          :attachment_record,
          space_record:,
          attached_space_member_record: space_member_record,
          active_storage_attachment_id: attachment1.id
        )

        # 2つ目の添付ファイル
        blob2 = ActiveStorage::Blob.create_and_upload!(
          io: StringIO.new("second image"),
          filename: "image.png",
          content_type: "image/png"
        )
        attachment2 = ActiveStorage::Attachment.create!(
          name: "file",
          record: space_record,
          blob: blob2
        )

        attachment_record2 = create(
          :attachment_record,
          space_record:,
          attached_space_member_record: space_member_record,
          active_storage_attachment_id: attachment2.id
        )

        # ページ本文に両方の添付ファイルへのリンクを含める
        page_body = <<~MARKDOWN
          ![First Image](/attachments/#{attachment_record1.id})
          ![Second Image](/attachments/#{attachment_record2.id})
        MARKDOWN

        page_record.update!(body: page_body)

        # 添付ファイル参照を作成
        PageAttachmentReferenceRecord.create!(
          page_id: page_record.id,
          attachment_id: attachment_record1.id
        )
        PageAttachmentReferenceRecord.create!(
          page_id: page_record.id,
          attachment_id: attachment_record2.id
        )

        # エクスポートレコードを作成
        export_record = create(:export_record, :queued,
          space_record:,
          queued_by_record: space_member_record)

        # ZIPファイルをテスト用に一時ディレクトリに作成
        Dir.mktmpdir do |temp_dir|
          export_dir = File.join(temp_dir, "export_#{export_record.id}")
          allow(Rails.root).to receive(:join).and_call_original
          allow(Rails.root).to receive(:join).with("tmp", "export_#{export_record.id}").and_return(Pathname.new(export_dir))

          # サービスを実行
          Spaces::GenerateExportFilesService.new.call(export_record:)

          # ZIPファイルが作成されていることを確認
          zip_path = "#{export_dir}.zip"

          Zip::File.open(zip_path) do |zipfile|
            # 両方のファイルが異なる名前で保存されていることを確認
            expect(zipfile.find_entry("attachments/image.png")).not_to be_nil
            expect(zipfile.find_entry("attachments/image_1.png")).not_to be_nil

            # ページ内容を確認
            page_entry = zipfile.find_entry("#{topic_record.name}/#{page_record.title}.md")
            page_content = zipfile.read(page_entry)

            # 相対パスに変換されていることを確認
            expect(page_content).to match(/!\[First Image\]\(attachments\/image(_\d+)?\.png\)/)
            expect(page_content).to match(/!\[Second Image\]\(attachments\/image(_\d+)?\.png\)/)
          end
        end
      end
    end
  end
end
