# typed: false
# frozen_string_literal: true

RSpec.describe Headers::GlobalComponent, type: :view do
  def render_header
    render_inline(described_class.new(current_page_name: PageName::Home, current_user: nil)) do |component|
      component.with_breadcrumb { "breadcrumb-content" }
    end
  end

  it "サイドバー開閉ボタン (basecoat:sidebar トグル) が描画されないこと" do
    result = render_header

    expect(result.to_html).not_to include("basecoat:sidebar")
    expect(result.css("button")).to be_empty
  end

  it "パンくずが描画されること" do
    result = render_header

    expect(result.to_html).to include("breadcrumb-content")
  end
end
