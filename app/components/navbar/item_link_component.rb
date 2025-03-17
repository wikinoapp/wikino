# typed: strict
# frozen_string_literal: true

module Navbar
  class ItemLinkComponent < ApplicationComponent
    sig { params(path: String, icon_name: String, title: String).void }
    def initialize(path:, icon_name:, title:)
      @path = path
      @icon_name = icon_name
      @title = title
    end

    sig { returns(String) }
    attr_reader :path
    private :path

    sig { returns(String) }
    attr_reader :icon_name
    private :icon_name

    sig { returns(String) }
    attr_reader :title
    private :title
  end
end
