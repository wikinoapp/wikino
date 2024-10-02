# typed: false
# frozen_string_literal: true

RSpec.describe Page, type: :model do
  describe "#titles_in_body" do
    context "トピックが存在するとき" do
      let!(:topic) { create(:topic) }

      it "returns titles" do
        [
          ["[[a]]", ["#{topic.name}/a"]],
          ["[[ a ]]", ["#{topic.name}/a"]],
          ["[[Hello]]", ["#{topic.name}/Hello"]],
          ["[[こんにちは✌️]]", ["#{topic.name}/こんにちは✌️"]],
          ["[[a]] [[b]]", ["#{topic.name}/a", "#{topic.name}/b"]],
          ["[[Hello]] [[World]]", ["#{topic.name}/Hello", "#{topic.name}/World"]],
          ["[[こんにちは]] [[世界🌏]]", ["#{topic.name}/こんにちは", "#{topic.name}/世界🌏"]],
          ["[ [a] ]", []],
          ["[[a]", []],
          # A bit weird, but same behavior as Obsidian, Reflect, Bear and etc.
          ["[[[a]]]", ["#{topic.name}/[a"]],
          ["[[[a]]] [[b]]", ["#{topic.name}/[a", "#{topic.name}/b"]],
          ["[[[a]]] [[[b]]]", ["#{topic.name}/[a", "#{topic.name}/[b"]],
          ["[[[ a ]]]", ["#{topic.name}/[ a"]]
        ].each do |(body, expected)|
          page = Page.new(space: topic.space, topic:, body:)
          expect(page.titles_in_body).to eq(expected)
        end
      end
    end
  end

  describe "#fetch_link_list" do
    context "記事にリンクが含まれているとき" do
      let!(:page_a) { create(:page, modified_at: Time.zone.parse("2024-01-01")) }
      let!(:page_b) { create(:page, modified_at: Time.zone.parse("2024-01-02")) }
      let!(:page_c) { create(:page, linked_page_ids: [page_b.id], modified_at: Time.zone.parse("2024-01-03")) }
      let!(:page_d) { create(:page, linked_page_ids: [page_c.id], modified_at: Time.zone.parse("2024-01-04")) }
      let!(:target_page) { create(:page, linked_page_ids: [page_a.id, page_c.id]) }

      it "リンクの構造体を返すこと" do
        link_list = target_page.fetch_link_list

        expect(link_list.links.size).to eq(2)

        link_a = link_list.links[0]
        expect(link_a.page).to eq(page_c)
        expect(link_a.backlinked_pages.size).to eq(1)
        expect(link_a.backlinked_pages[0]).to eq(page_d)

        link_b = link_list.links[1]
        expect(link_b.page).to eq(page_a)
        expect(link_b.backlinked_pages.size).to eq(0)
      end
    end
  end
end
