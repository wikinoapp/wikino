# typed: false
# frozen_string_literal: true

RSpec.describe Navbars::GlobalBottomComponent, type: :view do
  let(:user) do
    UserRepository.new.to_model(user_record: FactoryBot.create(:user_record, atname: "alice"))
  end

  it "md 未満でのみ表示する <nav> として描画されること" do
    result = render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

    expect(result.at_css("nav")["class"]).to include("md:hidden")
  end

  it "共通メニューを横並びの浮遊ピルとして描画すること" do
    result = render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

    menu = result.at_css("nav > div")
    expect(menu["class"]).to include("bg-card", "w-fit", "rounded-full", "border", "border-stone-400", "px-4", "py-1")
  end

  it "<nav> ランドマークに上部バーとは異なる固有の aria-label が付くこと" do
    result = render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

    expect(result.at_css("nav")["aria-label"]).to eq(I18n.t("messages.navbars.global_bottom_label"))
    expect(result.at_css("nav")["aria-label"]).not_to eq(I18n.t("messages.navbars.global_top_label"))
  end

  it "共通メニューのリンクを含むこと" do
    render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

    expect(page).to have_link(href: "/home")
  end
end
