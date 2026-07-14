# typed: strict
# frozen_string_literal: true

module Navbars
  # Mobile bottom bar, shown only below md. It carries the shared menu laid out
  # horizontally as a floating pill; the pill styling is passed to the menu so
  # the <nav> stays a bare landmark. The layout is responsible for fixing it to
  # the bottom of the screen, so this component only owns the visibility, the
  # landmark, and the pill.
  #
  # The wrapper is a <nav> landmark whose aria-label differs from the rail's,
  # distinguishing the two coexisting navigation regions (see GlobalRailComponent).
  #
  # [Ja] モバイル向けの下部バー。md 未満でのみ表示する。共通メニューを横並びの浮遊ピル
  # として持つ。ピルのスタイルはメニューに渡し、<nav> はランドマークに徹する。画面下部
  # への固定はレイアウトの責務のため、本コンポーネントは表示切り替え・ランドマーク・ピル
  # のみを担う。
  #
  # ラッパーは <nav> ランドマークで、その aria-label はレールとは異なり、共存する 2 つの
  # ナビゲーション領域を区別する (GlobalRailComponent を参照)。
  class GlobalBottomComponent < ApplicationComponent
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
