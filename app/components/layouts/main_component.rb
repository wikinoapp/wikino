# typed: strict
# frozen_string_literal: true

module Layouts
  class MainComponent < ApplicationComponent
    class ContentScreen < T::Enum
      enums do
        Small = new("sm")
        Medium = new("md")
        MediumLarge = new("md_lg")
        Large = new("lg")
      end
    end

    renders_one :breadcrumbs
    renders_one :main_content

    sig do
      params(
        signed_in: T::Boolean,
        current_page_name: PageName,
        content_screen: ContentScreen,
        class_name: String
      ).void
    end
    def initialize(signed_in:, current_page_name:, content_screen: ContentScreen::MediumLarge, class_name: "")
      @signed_in = signed_in
      @current_page_name = current_page_name
      @content_screen = content_screen
      @class_name = class_name
    end

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in
    alias_method :signed_in?, :signed_in

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    sig { returns(ContentScreen) }
    attr_reader :content_screen
    private :content_screen

    sig { returns(String) }
    attr_reader :class_name
    private :class_name

    sig { returns(String) }
    def content_screen_class_name
      cs = content_screen

      case cs
      when ContentScreen::Small
        "max-w-screen-sm"
      when ContentScreen::Medium
        "max-w-screen-md"
      when ContentScreen::MediumLarge
        "max-w-4xl"
      when ContentScreen::Large
        "max-w-screen-lg"
      else
        T.absurd(cs)
      end
    end
  end
end
