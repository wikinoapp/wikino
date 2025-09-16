# typed: strict
# frozen_string_literal: true

module BaseUI
  class ContainerComponent < ApplicationComponent
    class As < T::Enum
      enums do
        Div = new("div")
        Main = new("main")
        Section = new("section")
      end
    end

    class ContentScreen < T::Enum
      enums do
        Small = new("sm")
        Medium = new("md")
        Large = new("lg")
        Full = new("full")
      end
    end

    sig { params(as: As, content_screen: ContentScreen, options: T::Hash[Symbol, String]).void }
    def initialize(as: As::Div, content_screen: ContentScreen::Medium, options: {})
      @as = as
      @content_screen = content_screen
      @options = options
    end

    sig { returns(As) }
    attr_reader :as
    private :as

    sig { returns(ContentScreen) }
    attr_reader :content_screen
    private :content_screen

    sig { returns(T::Hash[Symbol, String]) }
    attr_reader :options
    private :options

    sig { returns(String) }
    private def max_width_class_name
      cs = content_screen

      case cs
      when ContentScreen::Small
        "max-w-(--content-screen-max-width-small)"
      when ContentScreen::Medium
        "max-w-(--content-screen-max-width-medium)"
      when ContentScreen::Large
        "max-w-(--content-screen-max-width-large)"
      when ContentScreen::Full
        "max-w-screen"
      else
        T.absurd(cs)
      end
    end

    sig { returns(String) }
    private def build_class_name
      class_names("mx-auto w-full pb-24", options[:class], max_width_class_name)
    end

    sig { returns(T::Hash[Symbol, String]) }
    private def build_options
      options.merge(
        class: build_class_name
      )
    end
  end
end
