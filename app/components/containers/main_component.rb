# typed: strict
# frozen_string_literal: true

module Containers
  class MainComponent < ApplicationComponent
    sig { params(content_screen: ContainerComponent::ContentScreen, class_name: String).void }
    def initialize(content_screen: ContainerComponent::ContentScreen::Medium, class_name: "")
      @content_screen = content_screen
      @class_name = class_name
    end

    sig { returns(ContainerComponent::ContentScreen) }
    attr_reader :content_screen
    private :content_screen

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
