# typed: false
# frozen_string_literal: true

RSpec.describe Markup, type: :model do
  def normalize_html(html)
    html.gsub(/\s+/, " ").strip
  end

  it "渡したテキストが空文字列のとき: 空文字列を返すこと" do
    topic = create(:topic)
    actual = Markup.new(current_topic: topic).render_html(text: "")

    expect(actual).to eq("")
  end

  it "タスクリスト記法: チェックボックスが生成されること" do
    topic = create(:topic)
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

  it "MarkupFilters::PageLinkFilter: リンク記法がリンクに置き換わること" do
    space = create(:space)
    topic_1 = create(:topic, space:, name: "トピック1")
    topic_2 = create(:topic, space:, name: "トピック2")
    page_1 = create(:page, space:, topic: topic_1, title: "Page 1")
    page_2 = create(:page, space:, topic: topic_2, title: "Page 2")
    page_3 = create(:page, space:, topic: topic_1, title: "Notebook -> List")

    text = <<~HTML
      - Test
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    expected = <<~HTML
      <ul>
        <li>Test</li>
      </ul>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))

    text = <<~HTML
      - [[Page 1]]
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    expected = <<~HTML
      <ul>
        <li><a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a></li>
      </ul>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))

    text = <<~HTML
      - [[トピック1/Page 1]]
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    expected = <<~HTML
      <ul>
        <li><a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a></li>
      </ul>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))

    text = <<~HTML
      - [[トピック1/Page 2]]
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    # Page 2はトピック2に属しているのでリンクにならないはず
    expected = <<~HTML
      <ul>
        <li>[[トピック1/Page 2]]</li>
      </ul>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))

    text = <<~HTML
      - [[Page 2]]
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    expected = <<~HTML
      <ul>
        <li>[[Page 2]]</li>
      </ul>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))

    text = <<~HTML
      - [[トピック2/Page 2]]
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    expected = <<~HTML
      <ul>
        <li><a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_2.number}">Page 2</a></li>
      </ul>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))

    text = <<~HTML
      - [[存在しないページ]]
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    expected = <<~HTML
      <ul>
        <li>[[存在しないページ]]</li>
      </ul>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))

    # `->` が含まれているとリンクにならないことがあったので追加 (リンクになるべき)
    text = <<~HTML
      - [[Notebook -> List]]
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    expected = <<~HTML
      <ul>
        <li><a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_3.number}">Notebook -&gt; List</a></li>
      </ul>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))

    text = <<~HTML
      文中にページリンクがある場合[[Page 1]]のテスト
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    expected = <<~HTML
      <p>文中にページリンクがある場合<a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a>のテスト</p>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))

    text = <<~HTML
      文中にトピック付きのページリンクがある場合[[トピック1/Page 1]]のテスト
    HTML
    actual = Markup.new(current_topic: topic_1).render_html(text:)
    expected = <<~HTML
      <p>文中にトピック付きのページリンクがある場合<a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a>のテスト</p>
    HTML
    expect(normalize_html(actual)).to eq(normalize_html(expected))
  end
end
