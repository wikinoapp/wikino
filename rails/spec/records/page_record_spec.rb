# typed: false
# frozen_string_literal: true

RSpec.describe PageRecord, type: :record do
  describe "#fetch_link_list" do
    it "ページにリンクが含まれているときリンクの構造体を返すこと" do
      user_record = create(:user_record)
      space_record = create(:space_record)
      space_record.reload
      topic_record = create(:topic_record, space_record:)
      create(:space_member_record, user_record:, space_record:)
      page_record_a = create(
        :page_record,
        space_record:,
        topic_record:,
        modified_at: Time.zone.parse("2024-01-01")
      )
      page_record_b = create(
        :page_record,
        space_record:,
        topic_record:,
        modified_at: Time.zone.parse("2024-01-02")
      )
      page_record_c = create(
        :page_record,
        space_record:,
        topic_record:,
        linked_page_ids: [page_record_b.id],
        modified_at: Time.zone.parse("2024-01-03")
      )
      page_record_d = create(
        :page_record,
        space_record:,
        topic_record:,
        linked_page_ids: [page_record_c.id],
        modified_at: Time.zone.parse("2024-01-04")
      )
      target_page_record = create(
        :page_record,
        space_record:,
        topic_record:,
        linked_page_ids: [page_record_a.id, page_record_c.id]
      )

      link_list = LinkListRepository.new.to_model(
        user_record:,
        pageable_record: target_page_record
      )
      expect(link_list.links.size).to eq(2)

      link_a = link_list.links[0]
      expect(link_a.backlink_list.backlinks.size).to eq(1)
      expect(link_a.backlink_list.backlinks[0]).to eq(
        Backlink.new(page: PageRepository.new.to_model(page_record: page_record_d, current_space_member: nil))
      )

      link_b = link_list.links[1]
      expect(link_b.page).to eq(PageRepository.new.to_model(page_record: page_record_a, current_space_member: nil))
      expect(link_b.backlink_list.backlinks.size).to eq(0)
    end
  end

  describe "#link!" do
    it "ページにリンクが含まれているとき、リンクを作成すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member_record = create(:space_member_record, user_record: user, space_record: space)

      topic_a = create(:topic_record, space_record: space, name: "トピックA")
      topic_b = create(:topic_record, space_record: space, name: "トピックB")
      page_a = create(:page_record, space_record: space, topic_record: topic_a, title: "Page A")

      expect(PageRecord.count).to eq(1)

      page_a.body = <<~BODY
        [[Page B]]
        [[トピックB/Page C]]
        [[存在しないトピック/Page D]]
      BODY
      page_a.link!(editor_record: space_member_record)

      expect(PageRecord.count).to eq(3)
      page_b = space.page_records.find_by(topic_record: topic_a, title: "Page B")
      expect(page_b).to be_present
      page_c = space.page_records.find_by(topic_record: topic_b, title: "Page C")
      expect(page_c).to be_present

      expect(page_a.linked_page_ids).to eq([page_b.id, page_c.id])
    end
  end

  describe "#update_attachment_references!" do
    it "HTMLタグから添付ファイルIDを抽出して参照を作成すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      # Active StorageとAttachmentRecordを作成
      blob1 = ActiveStorage::Blob.create!(
        key: "test_key_1",
        filename: "test1.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum1"
      )

      # ActiveStorage::AttachmentをSpaceRecordに関連付けて作成
      as_attachment1 = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob1
      )

      attachment1 = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment1.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      blob2 = ActiveStorage::Blob.create!(
        key: "test_key_2",
        filename: "test2.pdf",
        content_type: "application/pdf",
        metadata: {},
        byte_size: 2048,
        checksum: "checksum2"
      )

      as_attachment2 = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob2
      )

      attachment2 = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment2.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      # HTMLタグを含む本文
      body = <<~HTML
        <p>画像を表示します</p>
        <img src="/attachments/#{attachment1.id}" alt="テスト画像">
        <p>ファイルへのリンク</p>
        <a href="/attachments/#{attachment2.id}">PDFファイル</a>
      HTML

      # 参照を更新
      page.update_attachment_references!(body:)

      # 参照が作成されていることを確認
      references = PageAttachmentReferenceRecord.where(page_id: page.id)
      expect(references.count).to eq(2)
      expect(references.pluck(:attachment_id)).to contain_exactly(attachment1.id, attachment2.id)
    end

    it "Markdown形式から添付ファイルIDを抽出して参照を作成すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      # Active StorageとAttachmentRecordを作成
      blob1 = ActiveStorage::Blob.create!(
        key: "test_key_3",
        filename: "image.png",
        content_type: "image/png",
        metadata: {},
        byte_size: 3072,
        checksum: "checksum3"
      )

      as_attachment1 = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob1
      )

      attachment1 = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment1.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      blob2 = ActiveStorage::Blob.create!(
        key: "test_key_4",
        filename: "document.docx",
        content_type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        metadata: {},
        byte_size: 4096,
        checksum: "checksum4"
      )

      as_attachment2 = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob2
      )

      attachment2 = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment2.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      # Markdown形式を含む本文
      body = <<~MARKDOWN
        # ドキュメント

        画像を表示します:
        ![スクリーンショット](/attachments/#{attachment1.id})

        ドキュメントへのリンク:
        [ダウンロード](/attachments/#{attachment2.id})
      MARKDOWN

      # 参照を更新
      page.update_attachment_references!(body:)

      # 参照が作成されていることを確認
      references = PageAttachmentReferenceRecord.where(page_id: page.id)
      expect(references.count).to eq(2)
      expect(references.pluck(:attachment_id)).to contain_exactly(attachment1.id, attachment2.id)
    end

    it "存在しない添付ファイルIDは無視すること" do
      create(:user_record)
      space = create(:space_record)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      # 存在しないIDを含む本文
      body = <<~HTML
        <img src="/attachments/non-existent-id-1">
        ![画像](/attachments/non-existent-id-2)
      HTML

      # 参照を更新
      page.update_attachment_references!(body:)

      # 参照が作成されていないことを確認
      references = PageAttachmentReferenceRecord.where(page_id: page.id)
      expect(references.count).to eq(0)
    end

    it "既存の参照を更新して不要な参照を削除すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      # 3つのAttachmentRecordを作成
      attachments = 3.times.map do |i|
        blob = ActiveStorage::Blob.create!(
          key: "test_key_#{i}",
          filename: "file#{i}.jpg",
          content_type: "image/jpeg",
          metadata: {},
          byte_size: 1024 * (i + 1),
          checksum: "checksum#{i}"
        )

        as_attachment = ActiveStorage::Attachment.create!(
          name: "file",
          record: space,
          blob: blob
        )

        AttachmentRecord.create!(
          space_id: space.id,
          active_storage_attachment_id: as_attachment.id,
          attached_space_member_id: space_member.id,
          attached_at: Time.current,
          processing_status: AttachmentProcessingStatus::Completed.serialize
        )
      end

      # 最初は2つの添付ファイルを参照
      initial_body = <<~HTML
        <img src="/attachments/#{attachments[0].id}">
        <a href="/attachments/#{attachments[1].id}">リンク</a>
      HTML
      page.update_attachment_references!(body: initial_body)

      expect(PageAttachmentReferenceRecord.where(page_id: page.id).count).to eq(2)

      # 1つ目を削除して3つ目を追加
      updated_body = <<~HTML
        <a href="/attachments/#{attachments[1].id}">リンク</a>
        ![新しい画像](/attachments/#{attachments[2].id})
      HTML
      page.update_attachment_references!(body: updated_body)

      # 参照が正しく更新されていることを確認
      references = PageAttachmentReferenceRecord.where(page_id: page.id)
      expect(references.count).to eq(2)
      expect(references.pluck(:attachment_id)).to contain_exactly(attachments[1].id, attachments[2].id)
    end

    it "重複する添付ファイルIDは1つの参照のみ作成すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      # AttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_key",
        filename: "duplicate.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob
      )

      attachment = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      # 同じ添付ファイルを複数回参照
      body = <<~HTML
        <img src="/attachments/#{attachment.id}">
        <a href="/attachments/#{attachment.id}">リンク1</a>
        ![画像](/attachments/#{attachment.id})
        [リンク2](/attachments/#{attachment.id})
      HTML

      # 参照を更新
      page.update_attachment_references!(body:)

      # 参照は1つだけ作成されることを確認
      references = PageAttachmentReferenceRecord.where(page_id: page.id)
      expect(references.count).to eq(1)
      expect(references.first.attachment_id).to eq(attachment.id)
    end
  end

  describe "#extract_featured_image_id" do
    it "Markdown画像形式から画像IDを抽出すること" do
      space = create(:space_record)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      page.body = "![サムネイル画像](/attachments/abc-123-def)\n本文の続き"
      expect(page.extract_featured_image_id).to eq("abc-123-def")
    end

    it "HTML img要素から画像IDを抽出すること" do
      space = create(:space_record)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      page.body = "<img src=\"/attachments/xyz-456-uvw\" alt=\"画像\">\n本文の続き"
      expect(page.extract_featured_image_id).to eq("xyz-456-uvw")
    end

    it "1行目に画像がない場合はnilを返すこと" do
      space = create(:space_record)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      page.body = "テキストのみの1行目\n![画像](/attachments/abc-123)"
      expect(page.extract_featured_image_id).to be_nil
    end

    it "bodyが空の場合はnilを返すこと" do
      space = create(:space_record)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic, body: "")

      expect(page.extract_featured_image_id).to be_nil
    end

    it "1行目に前後の空白があっても画像IDを抽出すること" do
      space = create(:space_record)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      page.body = "  ![画像](/attachments/test-id-123)  \n続きの文章"
      expect(page.extract_featured_image_id).to eq("test-id-123")
    end
  end

  describe "#featured_image_is_gif?" do
    it "GIFファイルの場合はtrueを返すこと" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)

      # GIFファイルのAttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_gif",
        filename: "animation.gif",
        content_type: "image/gif",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum_gif"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob
      )

      attachment = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: attachment.id)
      expect(page.featured_image_is_gif?).to be true
    end

    it "GIF以外のファイルの場合はfalseを返すこと" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)

      # JPEGファイルのAttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_jpg",
        filename: "photo.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 2048,
        checksum: "checksum_jpg"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob
      )

      attachment = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: attachment.id)
      expect(page.featured_image_is_gif?).to be false
    end

    it "featured_image_attachment_recordがない場合はfalseを返すこと" do
      space = create(:space_record)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: nil)

      expect(page.featured_image_is_gif?).to be false
    end

    it "大文字の拡張子でもGIFと判定すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)

      # 大文字拡張子のGIFファイル
      blob = ActiveStorage::Blob.create!(
        key: "test_gif_upper",
        filename: "ANIMATION.GIF",
        content_type: "image/gif",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum_gif_upper"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob
      )

      attachment = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: attachment.id)
      expect(page.featured_image_is_gif?).to be true
    end
  end

  describe "#card_image_url" do
    it "通常の画像の場合はサムネイルURLを返すこと" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)

      # JPEGファイルのAttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_jpg_card",
        filename: "photo.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 2048,
        checksum: "checksum_jpg_card"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob
      )

      attachment = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: attachment.id)

      # reloadして関連を読み込み
      page.reload

      # thumbnail_urlが呼ばれることを期待
      allow(page.featured_image_attachment_record).to receive(:thumbnail_url).with(size: AttachmentThumbnailSize::Card, expires_in: 1.hour).and_return("https://example.com/thumbnail.jpg")
      expect(page.card_image_url).to eq("https://example.com/thumbnail.jpg")
    end

    it "GIFファイルの場合はオリジナル画像URLを返すこと" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)

      # GIFファイルのAttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_gif_card",
        filename: "animation.gif",
        content_type: "image/gif",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum_gif_card"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob
      )

      attachment = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: attachment.id)

      # reloadして関連を読み込み
      page.reload

      # generate_signed_urlが呼ばれることを期待
      allow(page.featured_image_attachment_record).to receive(:generate_signed_url).with(space_member_record: nil, expires_in: 1.hour).and_return("https://example.com/original.gif")
      expect(page.card_image_url).to eq("https://example.com/original.gif")
    end

    it "featured_image_attachment_recordがない場合はnilを返すこと" do
      space = create(:space_record)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: nil)

      expect(page.card_image_url).to be_nil
    end
  end

  describe "#og_image_url" do
    it "通常の画像の場合はOGサイズのサムネイルURLを返すこと" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)

      # JPEGファイルのAttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_jpg_og",
        filename: "photo.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 2048,
        checksum: "checksum_jpg_og"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob
      )

      attachment = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: attachment.id)

      # reloadして関連を読み込み
      page.reload

      # thumbnail_urlが呼ばれることを期待
      allow(page.featured_image_attachment_record).to receive(:thumbnail_url).with(size: AttachmentThumbnailSize::Og, expires_in: 1.hour).and_return("https://example.com/og-thumbnail.jpg")
      expect(page.og_image_url).to eq("https://example.com/og-thumbnail.jpg")
    end

    it "GIFファイルの場合はnilを返すこと" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)

      # GIFファイルのAttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_gif_og",
        filename: "animation.gif",
        content_type: "image/gif",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum_gif_og"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob
      )

      attachment = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: attachment.id)
      expect(page.og_image_url).to be_nil
    end

    it "featured_image_attachment_recordがない場合はnilを返すこと" do
      space = create(:space_record)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: nil)

      expect(page.og_image_url).to be_nil
    end
  end
end
