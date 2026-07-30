# typed: false
# frozen_string_literal: true

RSpec.describe Navbars::GlobalTopComponent, type: :view do
  let(:user) do
    UserRepository.new.to_model(user_record: FactoryBot.create(:user_record, atname: "alice"))
  end

  it "md 以上でのみ表示し、長いパンくずに潰されない <nav> として描画されること" do
    result = render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

    expect(result.at_css("nav")["class"]).to eq("shrink-0 hidden md:flex")
  end

  it "共通メニューをピルの装飾なしで描画すること" do
    result = render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

    menu = result.at_css("nav > div")
    expect(menu["class"]).to eq("flex items-center gap-2")
  end

  it "<nav> ランドマークに固有の aria-label が付くこと" do
    result = render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

    expect(result.at_css("nav")["aria-label"]).to eq(I18n.t("messages.navbars.global_top_label"))
  end

  it "共通メニューのリンクを含むこと" do
    render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

    expect(page).to have_link(href: "/home")
  end
end
