# typed: false
# frozen_string_literal: true

RSpec.describe Headers::GlobalComponent, type: :view do
  def render_header(with_breadcrumb: true, **args)
    render_inline(described_class.new(current_page_name: PageName::Home, current_user: nil, **args)) do |component|
      component.with_breadcrumb { "breadcrumb-content" } if with_breadcrumb
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

  it "本文と同じ幅のコンテナに、左のパンくずと右のナビバーを両端揃えで置くこと" do
    result = render_header

    container = result.at_css("header > div")
    expect(container["class"]).to eq(
      "max-w-(--content-screen-max-width-medium) mx-auto flex w-full items-center justify-between gap-2 px-4"
    )
    expect(container.at_css("> div")["class"]).to eq("min-w-0")
    expect(container.at_css("> nav")["class"]).to include("shrink-0")
  end

  it "中央揃え用の flex-1 スペーサーが描画されないこと" do
    result = render_header

    expect(result.css("header div.flex-1")).to be_empty
  end

  it "content_screen で指定した幅がコンテナに反映されること" do
    result = render_header(content_screen: BaseUI::ContainerComponent::ContentScreen::Small)

    expect(result.at_css("header > div")["class"]).to include("max-w-(--content-screen-max-width-small)")
  end

  it "グローバルナビの上部バーが描画されること" do
    result = render_header

    expect(result.at_css("nav[aria-label='#{I18n.t("messages.navbars.global_top_label")}']")).not_to be_nil
  end

  it "パンくずがある画面では <header> に表示切り替えクラスが付かないこと" do
    result = render_header

    expect(result.at_css("header")["class"]).to be_nil
  end

  it "パンくずがない画面では <header> をナビバーと同じ幅で切り替えること" do
    result = render_header(with_breadcrumb: false)

    expect(result.at_css("header")["class"]).to eq("hidden md:block")
    expect(result.at_css("nav[aria-label='#{I18n.t("messages.navbars.global_top_label")}']")["class"]).to include("md:flex")
  end
end
