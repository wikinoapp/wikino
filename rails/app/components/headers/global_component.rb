# typed: strict
# frozen_string_literal: true

module Headers
  class GlobalComponent < ApplicationComponent
    renders_one :breadcrumb

    sig do
      params(
        current_page_name: PageName,
        current_user: T.nilable(User),
        current_space: T.nilable(Space),
        content_screen: BaseUI::ContainerComponent::ContentScreen
      ).void
    end
    def initialize(current_page_name:, current_user:, current_space: nil, content_screen: BaseUI::ContainerComponent::ContentScreen::Medium)
      @current_page_name = current_page_name
      @current_user = current_user
      @current_space = current_space
      @content_screen = content_screen
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

    sig { returns(BaseUI::ContainerComponent::ContentScreen) }
    attr_reader :content_screen
    private :content_screen

    sig { returns(String) }
    private def max_width_class_name
      cs = content_screen

      case cs
      when BaseUI::ContainerComponent::ContentScreen::Small
        "max-w-(--content-screen-max-width-small)"
      when BaseUI::ContainerComponent::ContentScreen::Medium
        "max-w-(--content-screen-max-width-medium)"
      when BaseUI::ContainerComponent::ContentScreen::Large
        "max-w-(--content-screen-max-width-large)"
      when BaseUI::ContainerComponent::ContentScreen::Full
        "max-w-screen"
      else
        T.absurd(cs)
      end
    end
  end
end
