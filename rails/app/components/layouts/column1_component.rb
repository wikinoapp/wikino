# typed: strict
# frozen_string_literal: true

module Layouts
  # Single-column layout for authenticated pages (header / main / footer slots).
  #
  # [Ja] 認証後ページ向けの 1 カラムレイアウト (ヘッダー / メイン / フッターのスロットを持つ)。
  class Column1Component < ApplicationComponent
    renders_one :header
    renders_one :main
    renders_one :footer

    sig do
      params(
        current_page_name: PageName,
        current_user: T.nilable(User)
      ).void
    end
    def initialize(
      current_page_name:,
      current_user:
    )
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
