# typed: strict
# frozen_string_literal: true

module Navbars
  # Mobile bottom bar, shown only below md, where GlobalTopComponent takes over.
  # It carries the shared menu laid out horizontally as a floating pill; the pill
  # styling is passed to the menu so the <nav> stays a bare landmark. The layout
  # is responsible for fixing it to the bottom of the screen, so this component
  # only owns the visibility, the landmark, and the pill.
  #
  # The wrapper is a <nav> landmark whose aria-label differs from the top bar's,
  # distinguishing the two coexisting navigation regions (see GlobalTopComponent).
  #
  # [Ja] モバイル向けの下部バー。GlobalTopComponent に役割を譲る md 未満でのみ表示する。
  # 共通メニューを横並びの浮遊ピルとして持つ。ピルのスタイルはメニューに渡し、<nav> は
  # ランドマークに徹する。画面下部への固定はレイアウトの責務のため、本コンポーネントは
  # 表示切り替え・ランドマーク・ピルのみを担う。
  #
  # ラッパーは <nav> ランドマークで、その aria-label は上部バーとは異なり、共存する 2 つの
  # ナビゲーション領域を区別する (GlobalTopComponent を参照)。
  class GlobalBottomComponent < ApplicationComponent
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
