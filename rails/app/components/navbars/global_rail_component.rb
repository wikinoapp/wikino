# typed: strict
# frozen_string_literal: true

module Navbars
  # Desktop navigation rail: a floating pill fixed to the left edge and centered
  # vertically, shown only on md and up. It carries the shared menu laid out
  # vertically as a pill, mirroring GlobalBottomComponent so the rail and the
  # bottom bar share one visual language.
  #
  # The wrapper is a <nav> landmark with a distinct aria-label, paired with
  # GlobalBottomComponent's: the rail and the bottom bar are both rendered into
  # the same DOM (toggled by CSS breakpoints), so each carries its own label to
  # let assistive tech tell the two navigation regions apart.
  #
  # [Ja] PC 向けのナビゲーションレール。左端・縦中央に固定した浮遊ピルで、md 以上でのみ
  # 表示する。共通メニューを縦並びのピルとして持ち、GlobalBottomComponent と対になる
  # よう視覚言語を揃える。
  #
  # ラッパーは固有の aria-label を持つ <nav> ランドマークで、GlobalBottomComponent と
  # 対になる。レールと下部バーは CSS のブレークポイントで切り替わるが両方とも同一 DOM に
  # 描画されるため、支援技術が 2 つのナビゲーション領域を区別できるよう、それぞれに固有の
  # ラベルを付ける。
  class GlobalRailComponent < ApplicationComponent
    sig { params(current_page_name: PageName, current_user: T.nilable(User), current_space: T.nilable(Space)).void }
    def initialize(current_page_name:, current_user:, current_space: nil)
      @current_page_name = current_page_name
      @current_user = current_user
      @current_space = current_space
    end

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(T.nilable(User)) }
    attr_reader :current_user
    private :current_user

    sig { returns(T.nilable(Space)) }
    attr_reader :current_space
    private :current_space
  end
end
