# typed: false
# frozen_string_literal: true

RSpec.describe Page, type: :model do
  describe "#fetch_link_collection" do
    it "ページにリンクが含まれているときリンクの構造体を返すこと" do
      user = create(:user)
      space = create(:space)
      page_a = create(:page, space:, modified_at: Time.zone.parse("2024-01-01"))
      page_b = create(:page, space:, modified_at: Time.zone.parse("2024-01-02"))
      page_c = create(:page, space:, linked_page_ids: [page_b.id], modified_at: Time.zone.parse("2024-01-03"))
      page_d = create(:page, space:, linked_page_ids: [page_c.id], modified_at: Time.zone.parse("2024-01-04"))
      target_page = create(:page, space:, linked_page_ids: [page_a.id, page_c.id])

      Current.viewer = user
      link_collection = target_page.fetch_link_collection

      expect(link_collection.links.size).to eq(2)

      link_a = link_collection.links[0]
      expect(link_a.page).to eq(page_c)
      expect(link_a.backlink_collection.backlinks.size).to eq(1)
      expect(link_a.backlink_collection.backlinks[0]).to eql(Backlink.new(page: page_d))

      link_b = link_collection.links[1]
      expect(link_b.page).to eq(page_a)
      expect(link_b.backlink_collection.backlinks.size).to eq(0)
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

      expect(Page.count).to eq(1)

      page_a.body = <<~BODY
        [[Page B]]
        [[トピックB/Page C]]
        [[存在しないトピック/Page D]]
      BODY
      page_a.link!(editor: space_member)

      expect(Page.count).to eq(3)
      page_b = space.pages.find_by(topic: topic_a, title: "Page B")
      expect(page_b).to be_present
      page_c = space.pages.find_by(topic: topic_b, title: "Page C")
      expect(page_c).to be_present

      expect(page_a.linked_page_ids).to eq([page_b.id, page_c.id])
    end
  end
end
