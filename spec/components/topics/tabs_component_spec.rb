# typed: false
# frozen_string_literal: true

RSpec.describe Topics::TabsComponent, type: :view do
  it "タブが正しく表示されること" do
    tabs = [
      Topics::TabsComponent::TabItem.new(
        label: "ページ",
        path: "/test/pages",
        active: true
      ),
      Topics::TabsComponent::TabItem.new(
        label: "編集提案",
        path: "/test/edit_suggestions",
        active: false
      )
    ]

    render_inline(Topics::TabsComponent.new(tabs:))

    expect(page).to have_link("ページ", href: "/test/pages")
    expect(page).to have_link("編集提案", href: "/test/edit_suggestions")
  end

  it "アクティブなタブが正しくスタイリングされること" do
    tabs = [
      Topics::TabsComponent::TabItem.new(
        label: "ページ",
        path: "/test/pages",
        active: true
      ),
      Topics::TabsComponent::TabItem.new(
        label: "編集提案",
        path: "/test/edit_suggestions",
        active: false
      )
    ]

    render_inline(Topics::TabsComponent.new(tabs:))

    # アクティブなタブのスタイリングを確認
    active_link = page.find_link("ページ")
    expect(active_link[:class]).to include("border-primary-500")
    expect(active_link[:class]).to include("text-primary-600")

    # 非アクティブなタブのスタイリングを確認
    inactive_link = page.find_link("編集提案")
    expect(inactive_link[:class]).to include("border-transparent")
    expect(inactive_link[:class]).to include("text-gray-500")
  end

  it "タブナビゲーションが適切なHTML構造を持つこと" do
    tabs = [
      Topics::TabsComponent::TabItem.new(
        label: "ページ",
        path: "/test/pages",
        active: true
      )
    ]

    render_inline(Topics::TabsComponent.new(tabs:))

    expect(page).to have_css("nav")
    expect(page).to have_css("nav div.flex")
  end
end
