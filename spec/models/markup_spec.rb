# typed: false
# frozen_string_literal: true

RSpec.describe Markup, type: :model do
  def normalize_html(html)
    html.gsub(/\s+/, " ").strip
  end

  describe MarkupFilters::PageLinkFilter do
    it "リンク記法がリンクに置き換わること" do
      space = create(:space)
      topic_1 = create(:topic, space:, name: "トピック1")
      topic_2 = create(:topic, space:, name: "トピック2")

      create(:page, space:, topic: topic_1, title: "Page 1")
      create(:page, space:, topic: topic_2, title: "Page 2")

      actual = Markup.new(current_topic: topic_1).render_html(text: <<~MARKDOWN)
        - Test
        - [[Page 1]]
        - [[トピック1/Page 1]]
        - [[Page 2]]
        - [[トピック2/Page 2]]
        - [[存在しないページ]]
      MARKDOWN

      expected = <<~HTML
        <ul>
          <li>Test</li>
          <li>
            <a class="link link-primary" href="/s/identifier_1/pages/1">Page 1</a>
          </li>
          <li>
            <div class="flex">
              <a class="link link-primary" href="/s/identifier_1/topics/1">トピック1</a>
              <span>/</span>
              <a class="link link-primary" href="/s/identifier_1/pages/1">Page 1</a>
            </div>
          </li>
          <li>[[Page 2]]</li>
          <li>
            <div class="flex">
              <a class="link link-primary" href="/s/identifier_1/topics/2">トピック2</a>
              <span>/</span>
              <a class="link link-primary" href="/s/identifier_1/pages/2">Page 2</a>
            </div>
          </li>
          <li>[[存在しないページ]]</li>
        </ul>
      HTML

      expect(normalize_html(actual)).to eq(normalize_html(expected))
    end
  end
end
