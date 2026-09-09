# typed: strict
# frozen_string_literal: true

module Containers
  # Width-constrained container for the body of a screen, rendered into the
  # layout's main slot. It emits a plain <div> rather than a <main>: the single
  # main landmark of the page is the <main id="main"> that Layouts::Column1Component
  # renders around this slot, and it is also the target of the skip link.
  #
  # [Ja] 画面本文用の幅制限コンテナ。レイアウトの main スロットに描画される。
  # <main> ではなく素の <div> を出力する。ページで唯一の main ランドマークは
  # Layouts::Column1Component がこのスロットの外側に描画する <main id="main"> であり、
  # スキップリンクの飛び先でもある。
  class MainComponent < ApplicationComponent
    sig { params(content_screen: BaseUI::ContainerComponent::ContentScreen, class_name: String).void }
    def initialize(content_screen: BaseUI::ContainerComponent::ContentScreen::Medium, class_name: "")
      @content_screen = content_screen
      @class_name = class_name
    end

    sig { returns(BaseUI::ContainerComponent::ContentScreen) }
    attr_reader :content_screen
    private :content_screen

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
