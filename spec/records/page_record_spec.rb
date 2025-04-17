# typed: false
# frozen_string_literal: true

RSpec.describe PageRecord, type: :record do
  describe "#fetch_link_list_entity" do
    it "ページにリンクが含まれているときリンクの構造体を返すこと" do
      user_record = create(:user_record)
      space_record = create(:space_record)
      space_record.reload
      topic_record = create(:topic_record, space_record:)
      space_member_record = create(:space_member_record, user_record:, space_record:)
      page_record_a = create(:page_record,
        space_record:,
        topic_record:,
        modified_at: Time.zone.parse("2024-01-01")
      )
      page_record_b = create(:page_record,
        space_record:,
        topic_record:,
        modified_at: Time.zone.parse("2024-01-02")
      )
      page_record_c = create(:page_record,
        space_record:,
        topic_record:,
        linked_page_ids: [page_record_b.id],
        modified_at: Time.zone.parse("2024-01-03")
      )
      page_record_d = create(:page_record,
        space_record:,
        topic_record:,
        linked_page_ids: [page_record_c.id],
        modified_at: Time.zone.parse("2024-01-04")
      )
      target_page_record = create(:page_record,
        space_record:,
        topic_record:,
        linked_page_ids: [page_record_a.id, page_record_c.id]
      )

      Current.viewer = user_record

      link_list_entity = target_page_record.fetch_link_list_entity(space_viewer: space_member_record)
      expect(link_list_entity.link_entities.size).to eq(2)

      link_entity_a = link_list_entity.link_entities[0]
      expect(link_entity_a.backlink_list_entity.backlink_entities.size).to eq(1)
      expect(link_entity_a.backlink_list_entity.backlink_entities[0]).to eq(
        BacklinkEntity.new(page_entity: page_record_d.to_entity(space_viewer: space_member_record))
      )

      link_entity_b = link_list_entity.link_entities[1]
      expect(link_entity_b.page_entity).to eq(page_record_a.to_entity(space_viewer: space_member_record))
      expect(link_entity_b.backlink_list_entity.backlink_entities.size).to eq(0)
    end
  end

  describe "#link!" do
    it "ページにリンクが含まれているとき、リンクを作成すること" do
      user = create(:user_record)
      space = create(:space_record)
      space_member = create(:space_member_record, user_record: user, space_record: space)

      topic_a = create(:topic_record, space_record: space, name: "トピックA")
      topic_b = create(:topic_record, space_record: space, name: "トピックB")
      page_a = create(:page_record, space_record: space, topic_record: topic_a, title: "Page A")

      expect(PageRecord.count).to eq(1)

      page_a.body = <<~BODY
        [[Page B]]
        [[トピックB/Page C]]
        [[存在しないトピック/Page D]]
      BODY
      page_a.link!(editor: space_member)

      expect(PageRecord.count).to eq(3)
      page_b = space.page_records.find_by(topic_record: topic_a, title: "Page B")
      expect(page_b).to be_present
      page_c = space.page_records.find_by(topic_record: topic_b, title: "Page C")
      expect(page_c).to be_present

      expect(page_a.linked_page_ids).to eq([page_b.id, page_c.id])
    end
  end
end
