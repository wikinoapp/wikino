# typed: strict
# frozen_string_literal: true

module Navbars
  # Shared icon-only menu embedded by both the top bar and the bottom bar.
  # class_name is appended to the flex container so a wrapper can add its own
  # styling (the bottom bar uses it for the pill). current_page_name determines
  # the active item.
  #
  # Signed-in users see home / search / profile; signed-out users see home /
  # sign in. Search and profile require authentication and are omitted when
  # signed out.
  #
  # [Ja] 上部バーと下部バーの両方が埋め込む、アイコンのみの共通メニュー。class_name は
  # フレックスコンテナに追記され、ラッパーが固有のスタイルを足せるようにする (下部バーは
  # ピルの装飾に使う)。アクティブ項目は current_page_name で判定する。
  #
  # ログイン時はホーム / 検索 / プロフィール、未ログイン時はホーム / サインインを
  # 表示する。検索・プロフィールはログイン必須のため未ログイン時は表示しない。
  class GlobalMenuComponent < ApplicationComponent
    # Item is one navigation link's presentation data. label_key is an i18n key
    # resolved in the template (icon-only links get their accessible name from
    # the link's aria-label).
    #
    # [Ja] Item はナビゲーションリンク 1 件の表示用データ。label_key はテンプレートで
    # 解決する i18n キー (アイコンのみのリンクはリンクの aria-label でアクセシブル名を
    # 得る)。
    class Item < T::Struct
      const :path, String
      const :label_key, String
      const :default_icon, String
      const :active_icon, String
      const :active, T::Boolean
    end

    sig do
      params(
        current_page_name: PageName,
        current_user: T.nilable(User),
        class_name: String
      ).void
    end
    def initialize(current_page_name:, current_user:, class_name: "")
      @current_page_name = current_page_name
      @current_user = current_user
      @class_name = class_name
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(String) }
    attr_reader :class_name
    private :class_name

    sig { returns(String) }
    private def container_class_name
      ["flex items-center gap-2", class_name].compact_blank.join(" ")
    end

    # Colors the icon via the link's text color (the SVG picks it up through
    # fill-current). The active item is emphasized with the foreground color and
    # carries aria-current="page"; the link is fully rounded so its hover
    # highlight reads as a circle around the square icon.
    #
    # [Ja] リンクの text 色でアイコンを着色する (SVG は fill-current で継承)。
    # アクティブ項目は foreground 色で強調し aria-current="page" を付ける。リンクは
    # 完全な円 (rounded-full) にして、正方形アイコンを囲む hover ハイライトが円形に
    # 見えるようにする。
    sig { params(active: T::Boolean).returns(String) }
    private def item_class_name(active)
      base = "flex items-center justify-center rounded-full p-2 hover:bg-muted"
      color = active ? "text-foreground" : "text-muted-foreground hover:text-foreground"
      "#{base} #{color}"
    end

    sig { returns(T::Array[Item]) }
    private def items
      if signed_in?
        [
          Item.new(
            path: "/home",
            label_key: "nouns.home",
            default_icon: "house-regular",
            active_icon: "house-fill",
            active: current_page_name == PageName::Home
          ),
          Item.new(
            path: search_path,
            label_key: "nouns.search",
            default_icon: "magnifying-glass-regular",
            active_icon: "magnifying-glass-fill",
            active: current_page_name == PageName::Search
          ),
          Item.new(
            path: profile_path(current_user.not_nil!.atname),
            label_key: "nouns.profile",
            default_icon: "user-circle-regular",
            active_icon: "user-circle-fill",
            active: current_page_name == PageName::Profile
          )
        ]
      else
        [
          Item.new(
            path: "/",
            label_key: "nouns.home",
            default_icon: "house-regular",
            active_icon: "house-fill",
            active: current_page_name == PageName::Welcome
          ),
          Item.new(
            path: "/sign_in",
            label_key: "nouns.sign_in",
            default_icon: "sign-in-regular",
            active_icon: "sign-in-regular",
            active: false
          )
        ]
      end
    end

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user.nil?
    end
  end
end
