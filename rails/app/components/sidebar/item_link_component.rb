# typed: strict
# frozen_string_literal: true

module Sidebar
  class ItemLinkComponent < ApplicationComponent
    sig do
      params(
        path: String,
        title: String,
        default_icon_name: String,
        active_icon_name: String,
        active: T::Boolean
      ).void
    end
    def initialize(path:, title:, default_icon_name:, active_icon_name:, active:)
      @path = path
      @title = title
      @default_icon_name = default_icon_name
      @active_icon_name = active_icon_name
      @active = active
    end

    sig { returns(String) }
    attr_reader :path
    private :path

    sig { returns(String) }
    attr_reader :default_icon_name
    private :default_icon_name

    sig { returns(String) }
    attr_reader :active_icon_name
    private :active_icon_name

    sig { returns(T::Boolean) }
    attr_reader :active
    private :active

    sig { returns(String) }
    attr_reader :title
    private :title

    sig { returns(String) }
    private def icon_name
      active ? active_icon_name : default_icon_name
    end
  end
end
