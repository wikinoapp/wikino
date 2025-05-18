# typed: false
# frozen_string_literal: true

RSpec.describe SpaceService::Destroy, type: :service do
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

      SpaceService::Destroy.new.call(space_record_id: space_record.id)

      expect(SpaceRecord.count).to eq(0)
      expect(TopicRecord.count).to eq(0)
      expect(PageRecord.count).to eq(0)
      expect(DraftPageRecord.count).to eq(0)
      expect(PageEditorRecord.count).to eq(0)
      expect(PageRevisionRecord.count).to eq(0)
      expect(ExportRecord.count).to eq(0)
    end
  end
end
