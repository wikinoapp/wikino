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
    actual = Markup.new(current_topic: topic_1).render_html(
      text: [
        "- Test",
        "- [[Page 1]]",
        "- [[トピック1/Page 1]]",
        # Page 2はトピック2に属しているのでリンクにならないはず
        "- [[トピック1/Page 2]]",
        "- [[Page 2]]",
        "- [[トピック2/Page 2]]",
        "- [[存在しないページ]]",
        # `->` が含まれているとリンクにならないことがあったので追加 (リンクになるべき)
        "- [[Notebook -> List]]",
        "\n",
        "文中にページリンクがある場合[[Page 1]]のテスト",
        "\n",
        "文中にトピック付きのページリンクがある場合[[トピック1/Page 1]]のテスト"
      ].join("\n")
    )
    expected = <<~HTML
      <ul>
        <li>Test</li>
        <li><a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a></li>
        <li><a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a></li>
        <li>[[トピック1/Page 2]]</li>
        <li>[[Page 2]]</li>
        <li><a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_2.number}">Page 2</a></li>
        <li>[[存在しないページ]]</li>
        <li><a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_3.number}">Notebook -&gt; List</a></li>
      </ul>
      <p>文中にページリンクがある場合<a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a>のテスト</p>
      <p>文中にトピック付きのページリンクがある場合<a class="link link-primary" href="/s/#{space.identifier}/pages/#{page_1.number}">Page 1</a>のテスト</p>
    HTML

    expect(normalize_html(actual)).to eq(normalize_html(expected))
  end
end
