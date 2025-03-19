# typed: strict
# frozen_string_literal: true

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
    end
  end

  sig { params(as: As, content_screen: ContentScreen, class_name: String).void }
  def initialize(as: As::Div, content_screen: ContentScreen::Medium, class_name: "")
    @as = as
    @content_screen = content_screen
    @class_name = class_name
  end

  sig { returns(As) }
  attr_reader :as
  private :as

  sig { returns(ContentScreen) }
  attr_reader :content_screen
  private :content_screen

  sig { returns(String) }
  attr_reader :class_name
  private :class_name

  sig { returns(String) }
  def max_width_class_name
    cs = content_screen

    case cs
    when ContentScreen::Small
      "max-w-2xl" # 672px
    when ContentScreen::Medium
      "max-w-3xl" # 768px
    else
      T.absurd(cs)
    end
  end
end
