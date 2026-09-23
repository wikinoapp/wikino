# typed: false
# frozen_string_literal: true

RSpec.describe Layouts::Column1Component, type: :view do
  let(:user) do
    UserRepository.new.to_model(user_record: FactoryBot.create(:user_record, atname: "alice"))
  end

  def render_column1(current_user:)
    render_inline(described_class.new(current_page_name: PageName::Home, current_user:)) do |component|
      component.with_header { "header-content" }
      component.with_main { "main-content" }
    end
  end

  describe "グローバルナビの結線" do
    it "下部バーの <nav> ランドマークが描画されること" do
      result = render_column1(current_user: user)

      expect(result.css("nav[aria-label='#{I18n.t("messages.navbars.global_bottom_label")}']")).not_to be_empty
    end

    it "上部バーはレイアウトではなくヘッダーが描画すること" do
      result = render_column1(current_user: user)

      expect(result.css("nav[aria-label='#{I18n.t("messages.navbars.global_top_label")}']")).to be_empty
    end

    it "旧サイドバー (off-canvas ドロワー) が描画されないこと" do
      result = render_column1(current_user: user)

      expect(result.css("aside#sidebar")).to be_empty
    end
  end

  describe "スキップリンク" do
    it "本文へ飛ぶスキップリンクと main ランドマークが描画されること" do
      result = render_column1(current_user: user)

      skip_link = result.at_css("a[href='#main']")
      expect(skip_link).not_to be_nil
      expect(skip_link.text.strip).to eq(I18n.t("messages._common.skip_to_main_content"))
      expect(result.at_css("main#main[tabindex='-1']")).not_to be_nil
    end
  end
end
