# typed: false
# frozen_string_literal: true

RSpec.describe Page, type: :model do
  describe "#fetch_link_list_entity" do
    it "ページにリンクが含まれているときリンクの構造体を返すこと" do
      user = create(:user)
      space = create(:space)
      space.reload
      topic = create(:topic, space:)
      space_member = create(:space_member, user:, space:)
      page_a = create(:page, space:, topic:, modified_at: Time.zone.parse("2024-01-01"))
      page_b = create(:page, space:, topic:, modified_at: Time.zone.parse("2024-01-02"))
      page_c = create(:page, space:, topic:,
        linked_page_ids: [page_b.id],
        modified_at: Time.zone.parse("2024-01-03"))
      page_d = create(:page, space:, topic:,
        linked_page_ids: [page_c.id],
        modified_at: Time.zone.parse("2024-01-04"))
      target_page = create(:page, space:, topic:, linked_page_ids: [page_a.id, page_c.id])

      Current.viewer = user

      link_list_entity = target_page.fetch_link_list_entity(space_viewer: space_member)
      expect(link_list_entity.link_entities.size).to eq(2)

      link_entity_a = link_list_entity.link_entities[0]
      expect(link_entity_a.backlink_list_entity.backlink_entities.size).to eq(1)
      expect(link_entity_a.backlink_list_entity.backlink_entities[0]).to eq(
        BacklinkEntity.new(page_entity: page_d.to_entity(space_viewer: space_member))
      )

      link_entity_b = link_list_entity.link_entities[1]
      expect(link_entity_b.page_entity).to eq(page_a.to_entity(space_viewer: space_member))
      expect(link_entity_b.backlink_list_entity.backlink_entities.size).to eq(0)
    end
  end

  describe "#link!" do
    it "ページにリンクが含まれているとき、リンクを作成すること" do
      user = create(:user)
      space = create(:space)
      space_member = create(:space_member, user:, space:)

      topic_a = create(:topic, space:, name: "トピックA")
      topic_b = create(:topic, space:, name: "トピックB")
      page_a = create(:page, space:, topic: topic_a, title: "Page A")

      expect(PageRecord.count).to eq(1)

      page_a.body = <<~BODY
        [[Page B]]
        [[トピックB/Page C]]
        [[存在しないトピック/Page D]]
      BODY
      page_a.link!(editor: space_member)

      expect(PageRecord.count).to eq(3)
      page_b = space.pages.find_by(topic: topic_a, title: "Page B")
      expect(page_b).to be_present
      page_c = space.pages.find_by(topic: topic_b, title: "Page C")
      expect(page_c).to be_present

      expect(page_a.linked_page_ids).to eq([page_b.id, page_c.id])
    end
  end
end
