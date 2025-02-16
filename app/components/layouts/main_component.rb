# typed: strict
# frozen_string_literal: true

module Layouts
  class MainComponent < ApplicationComponent
    class ContentScreen < T::Enum
      enums do
        Small = new("sm")
        Medium = new("md")
        Large = new("lg")
      end
    end

    sig { params(current_page_name: PageName, content_screen: ContentScreen, class_name: String).void }
    def initialize(current_page_name:, content_screen: ContentScreen::Large, class_name: "")
      @current_page_name = current_page_name
      @content_screen = content_screen
      @class_name = class_name
    end

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
      when ContentScreen::Large
        "max-w-screen-lg"
      else
        T.absurd(cs)
      end
    end
  end
end
