# typed: false
# frozen_string_literal: true

RSpec.describe Page, type: :model do
  describe "#paths_in_body" do
    context "記事本文にリンク記法が書かれているとき" do
      let!(:page) { create(:page) }
      let!(:topic) { page.topic }

      it "パスのリストを返すこと" do
        [
          # 何も入力していないとき
          ["[[]]", []],
          # トピックを省略している場合
          ["[[a]]", [PagePath.new(topic_name: topic.name, page_title: "a")]],
          ["[[ a ]]", [PagePath.new(topic_name: topic.name, page_title: "a")]],
          ["[[Hello]]", [PagePath.new(topic_name: topic.name, page_title: "Hello")]],
          ["[[こんにちは✌️]]", [PagePath.new(topic_name: topic.name, page_title: "こんにちは✌️")]],
          ["[[a]] [[b]]", [
            PagePath.new(topic_name: topic.name, page_title: "a"),
            PagePath.new(topic_name: topic.name, page_title: "b")
          ]],
          ["[[Hello]] [[World]]", [
            PagePath.new(topic_name: topic.name, page_title: "Hello"),
            PagePath.new(topic_name: topic.name, page_title: "World")
          ]],
          ["[[こんにちは]] [[世界🌏]]", [
            PagePath.new(topic_name: topic.name, page_title: "こんにちは"),
            PagePath.new(topic_name: topic.name, page_title: "世界🌏")
          ]],
          ["[ [a] ]", []],
          ["[[a]", []],
          # A bit weird, but same behavior as Obsidian, Reflect, Bear and etc.
          ["[[[a]]]", [PagePath.new(topic_name: topic.name, page_title: "[a")]],
          ["[[[a]]] [[b]]", [
            PagePath.new(topic_name: topic.name, page_title: "[a"),
            PagePath.new(topic_name: topic.name, page_title: "b")
          ]],
          ["[[[a]]] [[[b]]]", [
            PagePath.new(topic_name: topic.name, page_title: "[a"),
            PagePath.new(topic_name: topic.name, page_title: "[b")
          ]],
          ["[[[ a ]]]", [PagePath.new(topic_name: topic.name, page_title: "[ a")]],

          # トピックを指定している場合
          ["[[foo/a]]", [PagePath.new(topic_name: "foo", page_title: "a")]],
          ["[[ foo/a ]]", [PagePath.new(topic_name: "foo", page_title: "a")]],
          ["[[ foo / a ]]", [PagePath.new(topic_name: "foo ", page_title: " a")]],
          ["[[foo/a]] [[bar/b]]", [
            PagePath.new(topic_name: "foo", page_title: "a"),
            PagePath.new(topic_name: "bar", page_title: "b")
          ]],
          ["[[foo/a/b]]", [PagePath.new(topic_name: "foo", page_title: "a/b")]]
        ].each do |(body, expected)|
          page.body = body

          expect(page.paths_in_body).to eq(expected)
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

  describe "#link!" do
    context "記事にリンクが含まれているとき" do
      let!(:space) { create(:space) }
      let!(:user) { create(:user, space:) }
      let!(:topic_a) { create(:topic, space:, name: "トピックA") }
      let!(:topic_b) { create(:topic, space:, name: "トピックB") }
      let!(:page_a) { create(:page, space:, topic: topic_a, title: "Page A") }

      it "リンクを作成すること" do
        expect(Page.count).to eq(1)

        page_a.body = <<~BODY
          [[Page B]]
          [[トピックB/Page C]]
          [[存在しないトピック/Page D]]
        BODY
        page_a.link!(editor: user)

        expect(Page.count).to eq(3)
        page_b = space.pages.find_by(topic: topic_a, title: "Page B")
        expect(page_b).to be_present
        page_c = space.pages.find_by(topic: topic_b, title: "Page C")
        expect(page_c).to be_present

        expect(page_a.linked_page_ids).to eq([page_b.id, page_c.id])
      end
    end
  end
end
