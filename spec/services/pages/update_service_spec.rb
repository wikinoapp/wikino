# typed: false
# frozen_string_literal: true

RSpec.describe Pages::UpdateService, type: :service do
  describe "#call" do
    it "ページを更新すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic, title: "Old Title", body: "Old Body")

      result = Pages::UpdateService.new.call(
        space_member_record: space_member,
        page_record: page,
        topic_record: topic,
        title: "New Title",
        body: "New Body"
      )

      expect(result.page_record.title).to eq("New Title")
      expect(result.page_record.body).to eq("New Body")
      expect(result.page_record.modified_at).to be_present
      expect(result.page_record.published_at).to be_present
    end

    it "1行目にMarkdown画像がある場合、featured_image_attachment_idを設定すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      # AttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_featured_image",
        filename: "featured.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum_featured"
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

      # 1行目に画像を含む本文
      body_with_image = "![サムネイル画像](/attachments/#{attachment.id})\n本文の続き"

      result = Pages::UpdateService.new.call(
        space_member_record: space_member,
        page_record: page,
        topic_record: topic,
        title: "ページタイトル",
        body: body_with_image
      )

      expect(result.page_record.featured_image_attachment_id).to eq(attachment.id)
    end

    it "1行目にHTML img要素がある場合、featured_image_attachment_idを設定すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      # AttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_featured_html",
        filename: "featured.png",
        content_type: "image/png",
        metadata: {},
        byte_size: 2048,
        checksum: "checksum_featured_html"
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

      # 1行目にHTML img要素を含む本文
      body_with_img = "<img src=\"/attachments/#{attachment.id}\" alt=\"画像\">\n本文の続き"

      result = Pages::UpdateService.new.call(
        space_member_record: space_member,
        page_record: page,
        topic_record: topic,
        title: "ページタイトル",
        body: body_with_img
      )

      expect(result.page_record.featured_image_attachment_id).to eq(attachment.id)
    end

    it "1行目に画像がない場合、featured_image_attachment_idをnilに設定すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)

      # 既にfeatured_image_attachment_idが設定されているページを作成
      # 実際に存在するAttachmentRecordを作成してIDを使用
      blob = ActiveStorage::Blob.create!(
        key: "test_old_featured",
        filename: "old.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum_old_featured"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space,
        blob: blob
      )

      old_attachment = AttachmentRecord.create!(
        space_id: space.id,
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: old_attachment.id)

      # 画像を含まない本文
      body_without_image = "テキストのみの本文\n2行目に続く"

      result = Pages::UpdateService.new.call(
        space_member_record: space_member,
        page_record: page,
        topic_record: topic,
        title: "ページタイトル",
        body: body_without_image
      )

      expect(result.page_record.featured_image_attachment_id).to be_nil
    end

    it "異なるスペースの画像IDの場合、featured_image_attachment_idをnilに設定すること" do
      user = create(:user_record)
      space1 = create(:space_record)
      space2 = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space1)
      topic = create(:topic_record, space_record: space1)
      page = create(:page_record, space_record: space1, topic_record: topic)

      # 異なるスペースのAttachmentRecordを作成
      blob = ActiveStorage::Blob.create!(
        key: "test_different_space",
        filename: "image.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum_different"
      )

      as_attachment = ActiveStorage::Attachment.create!(
        name: "file",
        record: space2,  # 異なるスペース
        blob: blob
      )

      other_space_member = create(:space_member_record, user_record: user, space_record: space2)
      attachment = AttachmentRecord.create!(
        space_id: space2.id,  # 異なるスペース
        active_storage_attachment_id: as_attachment.id,
        attached_space_member_id: other_space_member.id,
        attached_at: Time.current,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )

      # 異なるスペースの画像を参照する本文
      body_with_other_space_image = "![画像](/attachments/#{attachment.id})\n本文"

      result = Pages::UpdateService.new.call(
        space_member_record: space_member,
        page_record: page,
        topic_record: topic,
        title: "ページタイトル",
        body: body_with_other_space_image
      )

      expect(result.page_record.featured_image_attachment_id).to be_nil
    end

    it "存在しない画像IDの場合、featured_image_attachment_idをnilに設定すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      page = create(:page_record, space_record: space, topic_record: topic)

      # 存在しない画像IDを含む本文
      body_with_nonexistent_image = "![画像](/attachments/non-existent-id-123)\n本文"

      result = Pages::UpdateService.new.call(
        space_member_record: space_member,
        page_record: page,
        topic_record: topic,
        title: "ページタイトル",
        body: body_with_nonexistent_image
      )

      expect(result.page_record.featured_image_attachment_id).to be_nil
    end

    it "既存のfeatured画像を別の画像で更新すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)

      # 最初のAttachmentRecord
      blob1 = ActiveStorage::Blob.create!(
        key: "test_old_featured",
        filename: "old.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 1024,
        checksum: "checksum_old"
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

      # 新しいAttachmentRecord
      blob2 = ActiveStorage::Blob.create!(
        key: "test_new_featured",
        filename: "new.jpg",
        content_type: "image/jpeg",
        metadata: {},
        byte_size: 2048,
        checksum: "checksum_new"
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

      # 最初の画像を設定
      page = create(:page_record, space_record: space, topic_record: topic, featured_image_attachment_id: attachment1.id)

      # 新しい画像を含む本文で更新
      body_with_new_image = "![新しい画像](/attachments/#{attachment2.id})\n本文"

      result = Pages::UpdateService.new.call(
        space_member_record: space_member,
        page_record: page,
        topic_record: topic,
        title: "ページタイトル",
        body: body_with_new_image
      )

      expect(result.page_record.featured_image_attachment_id).to eq(attachment2.id)
    end

    it "topic_memberのlast_page_modified_atを更新すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      topic_member = create(:topic_member_record, 
        space_record: space,
        topic_record: topic,
        space_member_record: space_member,
        last_page_modified_at: nil
      )
      page = create(:page_record, space_record: space, topic_record: topic, title: "Old Title", body: "Old Body")

      result = Pages::UpdateService.new.call(
        space_member_record: space_member,
        page_record: page,
        topic_record: topic,
        title: "New Title",
        body: "New Body"
      )

      topic_member.reload
      expect(topic_member.last_page_modified_at).to be_present
      expect(topic_member.last_page_modified_at).to be_within(1.second).of(Time.current)
    end

    it "topic_memberが存在しない場合でもエラーにならないこと" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)
      topic = create(:topic_record, space_record: space)
      # topic_memberを作成しない
      page = create(:page_record, space_record: space, topic_record: topic, title: "Old Title", body: "Old Body")

      expect {
        result = Pages::UpdateService.new.call(
          space_member_record: space_member,
          page_record: page,
          topic_record: topic,
          title: "New Title",
          body: "New Body"
        )
        expect(result.page_record.title).to eq("New Title")
      }.not_to raise_error
    end
  end
end
