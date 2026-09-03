# typed: strict
# frozen_string_literal: true

module Navbars
  # Desktop navigation bar placed at the right end of the global header, shown
  # only on md and up, where GlobalBottomComponent hides itself. It carries the
  # shared menu laid out horizontally without a pill of its own: the bar shares
  # the header's row, so it sits in the normal flow and never overlays the
  # content. shrink-0 keeps the icons at full size when a long breadcrumb fills
  # the rest of the row.
  #
  # The wrapper is a <nav> landmark with a distinct aria-label, paired with
  # GlobalBottomComponent's: the top bar and the bottom bar are both rendered
  # into the same DOM (toggled by CSS breakpoints), so each carries its own
  # label to let assistive tech tell the two navigation regions apart.
  #
  # [Ja] PC 向けのナビゲーションバー。グローバルヘッダーの右端に置き、
  # GlobalBottomComponent が自身を隠す md 以上でのみ表示する。共通メニューを横並びで持ち、
  # 自身のピルは持たない。バーはヘッダーと同じ行を共有するため通常フローに入り、本文に
  # オーバーレイしない。長いパンくずが行を埋めてもアイコンが潰れないよう shrink-0 を付ける。
  #
  # ラッパーは固有の aria-label を持つ <nav> ランドマークで、GlobalBottomComponent と
  # 対になる。上部バーと下部バーは CSS のブレークポイントで切り替わるが両方とも同一 DOM に
  # 描画されるため、支援技術が 2 つのナビゲーション領域を区別できるよう、それぞれに固有の
  # ラベルを付ける。
  class GlobalTopComponent < ApplicationComponent
    sig { params(current_page_name: PageName, current_user: T.nilable(User)).void }
    def initialize(current_page_name:, current_user:)
      @current_page_name = current_page_name
      @current_user = current_user
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user
  end
end
