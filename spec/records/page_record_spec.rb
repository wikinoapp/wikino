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
        Backlink.new(page: PageRepository.new.to_model(page_record: page_record_d))
      )

      link_b = link_list.links[1]
      expect(link_b.page).to eq(PageRepository.new.to_model(page_record: page_record_a))
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
end
