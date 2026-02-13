# typed: false
# frozen_string_literal: true

RSpec.describe "Markup::PageLinkFilter", type: :model do
  def normalize_html(html)
    html.gsub(/\s+/, " ").strip
  end

  describe "リンク記法の変換" do
    it "通常のテキストはそのまま出力されること" do
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record: space_record, name: "トピック1")

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      text = "- Test"
      expected = "<ul>\n  <li>Test</li>\n</ul>"

      actual = Markup.new(current_topic: topic, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "同一トピック内のページリンクが変換されること" do
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record: space_record, name: "トピック1")
      page = create(:page_record, space_record: space_record, topic_record: topic_record, title: "Page 1")

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      text = "- [[Page 1]]"
      expected = "<ul>\n  <li><a href=\"/s/#{space_record.identifier}/pages/#{page.number}\">Page 1</a></li>\n</ul>"

      actual = Markup.new(current_topic: topic, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "トピック名付きの同一トピック内ページリンクが変換されること" do
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record: space_record, name: "トピック1")
      page = create(:page_record, space_record: space_record, topic_record: topic_record, title: "Page 1")

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      text = "- [[トピック1/Page 1]]"
      expected = "<ul>\n  <li><a href=\"/s/#{space_record.identifier}/pages/#{page.number}\">Page 1</a></li>\n</ul>"

      actual = Markup.new(current_topic: topic, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "トピック名が一致するが別トピックのページはリンクにならないこと" do
      space_record = create(:space_record)
      topic_one_record = create(:topic_record, space_record: space_record, name: "トピック1")
      topic_two_record = create(:topic_record, space_record: space_record, name: "トピック2")
      create(:page_record, space_record: space_record, topic_record: topic_two_record, title: "Page 2")

      topic_one = TopicRepository.new.to_model(topic_record: topic_one_record)
      space = SpaceRepository.new.to_model(space_record:)

      text = "- [[トピック1/Page 2]]"
      # Page 2はトピック2に属しているのでリンクにならないはず
      expected = "<ul>\n  <li>[[トピック1/Page 2]]</li>\n</ul>"

      actual = Markup.new(current_topic: topic_one, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "トピック名なしの別トピックページはリンクにならないこと" do
      space_record = create(:space_record)
      topic_one_record = create(:topic_record, space_record: space_record, name: "トピック1")
      topic_two_record = create(:topic_record, space_record: space_record, name: "トピック2")
      create(:page_record, space_record: space_record, topic_record: topic_two_record, title: "Page 2")

      topic_one = TopicRepository.new.to_model(topic_record: topic_one_record)
      space = SpaceRepository.new.to_model(space_record:)

      text = "- [[Page 2]]"
      expected = "<ul>\n  <li>[[Page 2]]</li>\n</ul>"

      actual = Markup.new(current_topic: topic_one, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "トピック名付きの別トピックページリンクが変換されること" do
      space_record = create(:space_record)
      topic_one_record = create(:topic_record, space_record: space_record, name: "トピック1")
      topic_two_record = create(:topic_record, space_record: space_record, name: "トピック2")
      page = create(:page_record, space_record: space_record, topic_record: topic_two_record, title: "Page 2")

      topic_one = TopicRepository.new.to_model(topic_record: topic_one_record)
      space = SpaceRepository.new.to_model(space_record:)

      text = "- [[トピック2/Page 2]]"
      expected = "<ul>\n  <li><a href=\"/s/#{space_record.identifier}/pages/#{page.number}\">Page 2</a></li>\n</ul>"

      actual = Markup.new(current_topic: topic_one, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "存在しないページはリンクにならないこと" do
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record: space_record, name: "トピック1")

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      text = "- [[存在しないページ]]"
      expected = "<ul>\n  <li>[[存在しないページ]]</li>\n</ul>"

      actual = Markup.new(current_topic: topic, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "特殊文字（->）を含むページタイトルのリンクが変換されること" do
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record: space_record, name: "トピック1")
      page = create(:page_record, space_record: space_record, topic_record: topic_record, title: "Notebook -> List")

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      text = "- [[Notebook -> List]]"
      expected = "<ul>\n  <li><a href=\"/s/#{space_record.identifier}/pages/#{page.number}\">Notebook -&gt; List</a></li>\n</ul>"

      actual = Markup.new(current_topic: topic, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "文中のページリンクが変換されること" do
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record: space_record, name: "トピック1")
      page = create(:page_record, space_record: space_record, topic_record: topic_record, title: "Page 1")

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      text = "文中にページリンクがある場合[[Page 1]]のテスト"
      expected = "<p>文中にページリンクがある場合<a href=\"/s/#{space_record.identifier}/pages/#{page.number}\">Page 1</a>のテスト</p>"

      actual = Markup.new(current_topic: topic, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "文中のトピック付きページリンクが変換されること" do
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record: space_record, name: "トピック1")
      page = create(:page_record, space_record: space_record, topic_record: topic_record, title: "Page 1")

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      text = "文中にトピック付きのページリンクがある場合[[トピック1/Page 1]]のテスト"
      expected = "<p>文中にトピック付きのページリンクがある場合<a href=\"/s/#{space_record.identifier}/pages/#{page.number}\">Page 1</a>のテスト</p>"

      actual = Markup.new(current_topic: topic, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "同じ行に複数のページリンクが変換されること" do
      space_record = create(:space_record)
      topic_one_record = create(:topic_record, space_record: space_record, name: "トピック1")
      topic_two_record = create(:topic_record, space_record: space_record, name: "トピック2")
      page_one = create(:page_record, space_record: space_record, topic_record: topic_one_record, title: "Page 1")
      page_two = create(:page_record, space_record: space_record, topic_record: topic_two_record, title: "Page 2")

      topic_one = TopicRepository.new.to_model(topic_record: topic_one_record)
      space = SpaceRepository.new.to_model(space_record:)

      text = "同じ行に2つのページリンクがある場合: [[Page 1]] [[トピック2/Page 2]]"
      expected = "<p>同じ行に2つのページリンクがある場合: <a href=\"/s/#{space_record.identifier}/pages/#{page_one.number}\">Page 1</a> <a href=\"/s/#{space_record.identifier}/pages/#{page_two.number}\">Page 2</a></p>"

      actual = Markup.new(current_topic: topic_one, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end

    it "正規表現の特殊文字（括弧）を含むページタイトルのリンクが変換されること" do
      space_record = create(:space_record)
      topic_record = create(:topic_record, space_record: space_record, name: "トピック1")
      page = create(:page_record, space_record: space_record, topic_record: topic_record, title: "日記 (2025)")

      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record:)

      text = "[[日記 (2025)]]"
      expected = "<p><a href=\"/s/#{space_record.identifier}/pages/#{page.number}\">日記 (2025)</a></p>"

      actual = Markup.new(current_topic: topic, current_space: space).render_html(text: text)
      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end
  end
end
