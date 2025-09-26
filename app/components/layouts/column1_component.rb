# typed: strict
# frozen_string_literal: true

module Layouts
  # サイドバーを除いた1カラムのレイアウト
  class Column1Component < ApplicationComponent
    renders_one :header
    renders_one :main
    renders_one :footer

    sig do
      params(
        current_page_name: PageName,
        current_user: T.nilable(User),
        current_space: T.nilable(Space),
        show_sidebar: T::Boolean,
        show_bottom_navbar: T::Boolean,
        controller_name: String
      ).void
    end
    def initialize(
      current_page_name:,
      current_user:,
      current_space: nil,
      show_sidebar: true,
      show_bottom_navbar: true,
      controller_name: ""
    )
      @current_page_name = current_page_name
      @current_user = current_user
      @current_space = current_space
      @show_sidebar = show_sidebar
      @show_bottom_navbar = show_bottom_navbar
      @controller_name = controller_name
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

    sig { returns(T::Boolean) }
    attr_reader :show_sidebar
    alias_method :show_sidebar?, :show_sidebar

    sig { returns(T::Boolean) }
    attr_reader :show_bottom_navbar
    private :show_bottom_navbar
    alias_method :show_bottom_navbar?, :show_bottom_navbar

    sig { returns(String) }
    attr_reader :controller_name
    private :controller_name

    sig { returns(String) }
    def controller_names
      "global-hotkey sidebar #{controller_name}".strip
    end
  end
end
