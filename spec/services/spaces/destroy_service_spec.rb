# typed: false
# frozen_string_literal: true

RSpec.describe Spaces::DestroyService, type: :service do
  describe "#call" do
    it "スペースを削除できること" do
      space_record = create(:space_record)
      space_member_record = create(:space_member_record, space_record:)
      topic_record = create(:topic_record, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, space_record:, topic_record:)
      create(:draft_page_record, space_record:, page_record:, space_member_record:, topic_record:)
      create(:page_editor_record, space_record:, page_record:, space_member_record:)
      create(:page_revision_record, space_record:, page_record:, space_member_record:)
      create(:export_record, :succeeded, space_record:, queued_by_record: space_member_record)

      expect(SpaceRecord.count).to eq(1)
      expect(TopicRecord.count).to eq(1)
      expect(PageRecord.count).to eq(1)
      expect(DraftPageRecord.count).to eq(1)
      expect(PageEditorRecord.count).to eq(1)
      expect(PageRevisionRecord.count).to eq(1)
      expect(ExportRecord.count).to eq(1)

      Spaces::DestroyService.new.call(space_record_id: space_record.id)

      expect(SpaceRecord.count).to eq(0)
      expect(TopicRecord.count).to eq(0)
      expect(PageRecord.count).to eq(0)
      expect(DraftPageRecord.count).to eq(0)
      expect(PageEditorRecord.count).to eq(0)
      expect(PageRevisionRecord.count).to eq(0)
      expect(ExportRecord.count).to eq(0)
    end

    it "添付ファイルを含むスペースを削除できること" do
      space_record = create(:space_record)
      space_member_record = create(:space_member_record, space_record:)
      topic_record = create(:topic_record, space_record:)
      create(:topic_member_record, space_record:, topic_record:, space_member_record:)
      page_record = create(:page_record, space_record:, topic_record:)
      create(:draft_page_record, space_record:, page_record:, space_member_record:, topic_record:)
      create(:page_editor_record, space_record:, page_record:, space_member_record:)
      create(:page_revision_record, space_record:, page_record:, space_member_record:)
      create(:export_record, :succeeded, space_record:, queued_by_record: space_member_record)

      # 添付ファイルを作成
      attachment_record_1 = create(:attachment_record, :with_blob, space_record:, attached_space_member_record: space_member_record)
      attachment_record_2 = create(:attachment_record, :with_blob, space_record:, attached_space_member_record: space_member_record)
      create(:attachment_record, :with_blob, :processing, space_record:, attached_space_member_record: space_member_record)

      # 添付ファイルの参照を作成
      create(:page_attachment_reference_record, page_record:, attachment_record: attachment_record_1)
      create(:page_attachment_reference_record, page_record:, attachment_record: attachment_record_2)

      expect(SpaceRecord.count).to eq(1)
      expect(TopicRecord.count).to eq(1)
      expect(PageRecord.count).to eq(1)
      expect(DraftPageRecord.count).to eq(1)
      expect(PageEditorRecord.count).to eq(1)
      expect(PageRevisionRecord.count).to eq(1)
      expect(ExportRecord.count).to eq(1)
      expect(AttachmentRecord.count).to eq(3)
      expect(PageAttachmentReferenceRecord.count).to eq(2)
      expect(ActiveStorage::Attachment.count).to eq(3)
      expect(ActiveStorage::Blob.count).to eq(3)

      Spaces::DestroyService.new.call(space_record_id: space_record.id)

      expect(SpaceRecord.count).to eq(0)
      expect(TopicRecord.count).to eq(0)
      expect(PageRecord.count).to eq(0)
      expect(DraftPageRecord.count).to eq(0)
      expect(PageEditorRecord.count).to eq(0)
      expect(PageRevisionRecord.count).to eq(0)
      expect(ExportRecord.count).to eq(0)
      expect(AttachmentRecord.count).to eq(0)
      expect(PageAttachmentReferenceRecord.count).to eq(0)
      # Active StorageのAttachmentとBlobもカスケード削除されることを確認
      expect(ActiveStorage::Attachment.count).to eq(0)
      expect(ActiveStorage::Blob.count).to eq(0)
    end
  end
end
