# typed: strict
# frozen_string_literal: true

module Containers
  class MainComponent < ApplicationComponent
    class ContentScreen < T::Enum
      enums do
        Small = new("sm")
        Medium = new("md")
        Large = new("lg")
      end
    end

    sig { params(content_screen: ContentScreen).void }
    def initialize(content_screen: ContentScreen::Medium)
      @content_screen = content_screen
    end

    sig { returns(ContentScreen) }
    attr_reader :content_screen
    private :content_screen

    sig { returns(String) }
    def max_width_class_name
      cs = content_screen

      case cs
      when ContentScreen::Small
        "max-w-2xl" # 672px
      when ContentScreen::Medium
        "max-w-4xl" # 896px
      when ContentScreen::Large
        "max-w-5xl" # 1024px
      else
        T.absurd(cs)
      end
    end
  end
end
