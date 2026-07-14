# typed: false
# frozen_string_literal: true

RSpec.describe Navbars::GlobalMenuComponent, type: :view do
  let(:user) do
    UserRepository.new.to_model(user_record: FactoryBot.create(:user_record, atname: "alice"))
  end

  let(:space) do
    SpaceRepository.new.to_model(space_record: FactoryBot.create(:space_record, identifier: "my-space"))
  end

  describe "ログイン時" do
    it "ホーム・検索・プロフィールのリンクが表示され、サインインリンクは表示されないこと" do
      render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

      expect(page).to have_link(href: "/home")
      expect(page).to have_link(href: "/search")
      expect(page).to have_link(href: "/@alice")
      expect(page).to have_no_link(href: "/sign_in")
    end

    it "3 項目それぞれがアイコンを 1 つずつ描画すること" do
      render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

      expect(page).to have_css("svg", count: 3)
    end

    it "各リンクに aria-label が付与されること" do
      render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

      expect(page).to have_css("a[href='/home'][aria-label='#{I18n.t("nouns.home")}']")
      expect(page).to have_css("a[href='/search'][aria-label='#{I18n.t("nouns.search")}']")
      expect(page).to have_css("a[href='/@alice'][aria-label='#{I18n.t("nouns.profile")}']")
    end
  end

  describe "未ログイン時" do
    it "ホーム (ルート) とサインインのみが表示されること" do
      render_inline(described_class.new(current_page_name: PageName::Welcome, current_user: nil))

      expect(page).to have_link(href: "/")
      expect(page).to have_link(href: "/sign_in")
      expect(page).to have_no_link(href: "/search")
      expect(page).to have_no_css("a[href^='/@']")
    end

    it "2 項目それぞれがアイコンを 1 つずつ描画すること" do
      render_inline(described_class.new(current_page_name: PageName::Welcome, current_user: nil))

      expect(page).to have_css("svg", count: 2)
    end

    it "ホームリンクとサインインリンクに aria-label が付与されること" do
      render_inline(described_class.new(current_page_name: PageName::Welcome, current_user: nil))

      expect(page).to have_css("a[href='/'][aria-label='#{I18n.t("nouns.home")}']")
      expect(page).to have_css("a[href='/sign_in'][aria-label='#{I18n.t("nouns.sign_in")}']")
    end
  end

  describe "アクティブ表示" do
    [
      [PageName::Home, "/home"],
      [PageName::Search, "/search"],
      [PageName::Profile, "/@alice"]
    ].each do |page_name, active_href|
      it "#{page_name.serialize} のとき #{active_href} だけがアクティブになること" do
        render_inline(described_class.new(current_page_name: page_name, current_user: user))

        expect(page).to have_css("a[aria-current='page']", count: 1)
        expect(page).to have_css("a[href='#{active_href}'][aria-current='page']")
      end
    end

    it "どの項目にも対応しないページ (トピック詳細など) ではアクティブ項目が無いこと" do
      render_inline(described_class.new(current_page_name: PageName::TopicDetail, current_user: user))

      expect(page).to have_no_css("a[aria-current='page']")
    end
  end

  describe "アクティブアイコンの切り替え" do
    it "アクティブ時は塗りつぶし、非アクティブ時は通常のアイコンを描画すること" do
      fill_d = render_inline(BaseUI::IconComponent.new(name: "house-fill", size: "24px")).css("path").first["d"]
      regular_d = render_inline(BaseUI::IconComponent.new(name: "house-regular", size: "24px")).css("path").first["d"]

      active = render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))
      inactive = render_inline(described_class.new(current_page_name: PageName::Search, current_user: user))

      expect(active.css("a[href='/home'] path").first["d"]).to eq(fill_d)
      expect(inactive.css("a[href='/home'] path").first["d"]).to eq(regular_d)
    end
  end

  describe "着色" do
    it "アイコンが fill-current でリンクの text 色を継承し、アクティブ / 非アクティブで色クラスが切り替わること" do
      result = render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

      active_link = result.at_css("a[href='/home']")
      inactive_link = result.at_css("a[href='/search']")

      expect(active_link.at_css("svg")["class"]).to include("fill-current")
      expect(active_link["class"]).to include("text-foreground")
      expect(inactive_link["class"]).to include("text-muted-foreground")
    end
  end

  describe "検索パス" do
    it "スペース内では検索リンクがそのスペースに絞り込まれること" do
      render_inline(described_class.new(current_page_name: PageName::Home, current_user: user, current_space: space))

      expect(page).to have_css("a[href*='space%3Amy-space']")
    end

    it "スペース外では素の検索パスになること" do
      render_inline(described_class.new(current_page_name: PageName::Home, current_user: user))

      expect(page).to have_link(href: "/search")
    end
  end

  describe "コンテナクラス" do
    it "渡した class_name がフレックスコンテナに追記されること" do
      result = render_inline(
        described_class.new(current_page_name: PageName::Home, current_user: user, class_name: "flex-col rounded-full")
      )

      expect(result.at_css("div")["class"]).to include("flex", "items-center", "gap-2", "flex-col", "rounded-full")
    end
  end
end
