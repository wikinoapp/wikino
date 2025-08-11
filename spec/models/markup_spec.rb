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
end
