# typed: false
# frozen_string_literal: true

RSpec.describe Markup, type: :model do
  def normalize_html(html)
    html.gsub(/\s+/, " ").strip
  end

  def test_render_html(current_topic:, text:, expected:)
    actual = Markup.new(current_topic:).render_html(text:)
    expect(normalize_html(actual)).to eq(normalize_html(expected))
  end

  it "渡したテキストが空文字列のとき: 空文字列を返すこと" do
    topic = create(:topic_record)
    actual = Markup.new(current_topic: topic).render_html(text: "")

    expect(actual).to eq("")
  end

  it "タスクリスト記法: チェックボックスが生成されること" do
    topic = create(:topic_record)
    actual = Markup.new(current_topic: topic).render_html(
      text: [
        "- [ ] 未完了",
        "- [x] 完了"
      ].join("\n")
    )
    expected = <<~HTML
      <ul>
        <li><input type="checkbox" disabled="" /> 未完了</li>
        <li><input type="checkbox" checked="" disabled="" /> 完了</li>
      </ul>
    HTML

    expect(normalize_html(actual)).to eq(normalize_html(expected))
  end

  it "Markup::PageLinkFilter: リンク記法がリンクに置き換わること" do # standard:disable RSpec/NoExpectationExample
    space = create(:space_record)
    topic_1 = create(:topic_record, space_record: space, name: "トピック1")
    topic_2 = create(:topic_record, space_record: space, name: "トピック2")
    page_1 = create(:page_record, space_record: space, topic_record: topic_1, title: "Page 1")
    page_2 = create(:page_record, space_record: space, topic_record: topic_2, title: "Page 2")
    page_3 = create(:page_record, space_record: space, topic_record: topic_1, title: "Notebook -> List")
    page_4 = create(:page_record, space_record: space, topic_record: topic_1, title: "日記 (2025)")

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        - Test
      TEXT
      expected: <<~HTML
        <ul>
          <li>Test</li>
        </ul>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        - [[Page 1]]
      TEXT
      expected: <<~HTML
        <ul>
          <li><a href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a></li>
        </ul>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        - [[トピック1/Page 1]]
      TEXT
      expected: <<~HTML
        <ul>
          <li><a href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a></li>
        </ul>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        - [[トピック1/Page 2]]
      TEXT
      # Page 2はトピック2に属しているのでリンクにならないはず
      expected: <<~HTML
        <ul>
          <li>[[トピック1/Page 2]]</li>
        </ul>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        - [[Page 2]]
      TEXT
      expected: <<~HTML
        <ul>
          <li>[[Page 2]]</li>
        </ul>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        - [[トピック2/Page 2]]
      TEXT
      expected: <<~HTML
        <ul>
          <li><a href="/s/#{space.identifier}/pages/#{page_2.number}">Page 2</a></li>
        </ul>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        - [[存在しないページ]]
      TEXT
      expected: <<~HTML
        <ul>
          <li>[[存在しないページ]]</li>
        </ul>
      HTML
    )

    # `->` が含まれているとリンクにならないことがあったので追加 (リンクになるべき)
    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        - [[Notebook -> List]]
      TEXT
      expected: <<~HTML
        <ul>
          <li><a href="/s/#{space.identifier}/pages/#{page_3.number}">Notebook -&gt; List</a></li>
        </ul>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        文中にページリンクがある場合[[Page 1]]のテスト
      TEXT
      expected: <<~HTML
        <p>文中にページリンクがある場合<a href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a>のテスト</p>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        文中にトピック付きのページリンクがある場合[[トピック1/Page 1]]のテスト
      TEXT
      expected: <<~HTML
        <p>文中にトピック付きのページリンクがある場合<a href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a>のテスト</p>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      text: <<~TEXT,
        同じ行に2つのページリンクがある場合: [[Page 1]] [[トピック2/Page 2]]
      TEXT
      expected: <<~HTML
        <p>同じ行に2つのページリンクがある場合: <a href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a> <a href="/s/#{space.identifier}/pages/#{page_2.number}">Page 2</a></p>
      HTML
    )

    test_render_html(
      current_topic: topic_1,
      # 正規表現において特別な意味を持つ文字がリンク記法内に含まれているとき
      text: <<~TEXT,
        [[日記 (2025)]]
      TEXT
      expected: <<~HTML
        <p><a href="/s/#{space.identifier}/pages/#{page_4.number}">日記 (2025)</a></p>
      HTML
    )
  end
end
