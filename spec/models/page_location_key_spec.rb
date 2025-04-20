# typed: false
# frozen_string_literal: true

RSpec.describe PageLocationKey, type: :model do
  describe ".scan_text" do
    context "テキスト内にリンク記法が書かれているとき" do
      let!(:current_topic) { create(:topic) }

      it "キーのリストを返すこと" do
        [
          # 何も入力していないとき
          ["[[]]", []],
          # トピックを省略している場合
          ["[[a]]", [PageLocationKey.new(raw: "a", topic_name: current_topic.name, page_title: "a")]],
          ["[[ a ]]", [PageLocationKey.new(raw: "a", topic_name: current_topic.name, page_title: "a")]],
          ["[[Hello]]", [PageLocationKey.new(raw: "Hello", topic_name: current_topic.name, page_title: "Hello")]],
          ["[[こんにちは✌️]]", [PageLocationKey.new(raw: "こんにちは✌️", topic_name: current_topic.name, page_title: "こんにちは✌️")]],
          ["[[a]] [[b]]", [
            PageLocationKey.new(raw: "a", topic_name: current_topic.name, page_title: "a"),
            PageLocationKey.new(raw: "b", topic_name: current_topic.name, page_title: "b")
          ]],
          ["[[Hello]] [[World]]", [
            PageLocationKey.new(raw: "Hello", topic_name: current_topic.name, page_title: "Hello"),
            PageLocationKey.new(raw: "World", topic_name: current_topic.name, page_title: "World")
          ]],
          ["[[こんにちは]] [[世界🌏]]", [
            PageLocationKey.new(raw: "こんにちは", topic_name: current_topic.name, page_title: "こんにちは"),
            PageLocationKey.new(raw: "世界🌏", topic_name: current_topic.name, page_title: "世界🌏")
          ]],
          ["[ [a] ]", []],
          ["[[a]", []],
          # A bit weird, but same behavior as Obsidian, Reflect, Bear and etc.
          ["[[[a]]]", [PageLocationKey.new(raw: "[a", topic_name: current_topic.name, page_title: "[a")]],
          ["[[[a]]] [[b]]", [
            PageLocationKey.new(raw: "[a", topic_name: current_topic.name, page_title: "[a"),
            PageLocationKey.new(raw: "b", topic_name: current_topic.name, page_title: "b")
          ]],
          ["[[[a]]] [[[b]]]", [
            PageLocationKey.new(raw: "[a", topic_name: current_topic.name, page_title: "[a"),
            PageLocationKey.new(raw: "[b", topic_name: current_topic.name, page_title: "[b")
          ]],
          ["[[[ a ]]]", [PageLocationKey.new(raw: "[ a", topic_name: current_topic.name, page_title: "[ a")]],

          # トピックを指定している場合
          ["[[foo/a]]", [PageLocationKey.new(raw: "foo/a", topic_name: "foo", page_title: "a")]],
          ["[[ foo/a ]]", [PageLocationKey.new(raw: "foo/a", topic_name: "foo", page_title: "a")]],
          ["[[ foo / a ]]", [PageLocationKey.new(raw: "foo / a", topic_name: "foo ", page_title: " a")]],
          ["[[foo/a]] [[bar/b]]", [
            PageLocationKey.new(raw: "foo/a", topic_name: "foo", page_title: "a"),
            PageLocationKey.new(raw: "bar/b", topic_name: "bar", page_title: "b")
          ]],
          ["[[foo/a/b]]", [PageLocationKey.new(raw: "foo/a/b", topic_name: "foo", page_title: "a/b")]]
        ].each do |(text, expected)|
          expect(PageLocationKey.scan_text(text:, current_topic:)).to eq(expected)
        end
      end
    end
  end
end
