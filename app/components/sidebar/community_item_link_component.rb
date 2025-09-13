# typed: strict
# frozen_string_literal: true

module Sidebar
  class CommunityItemLinkComponent < ApplicationComponent
    sig do
      params(
        url: String,
        title: String,
        icon_name: String,
        fill_class: String
      ).void
    end
    def initialize(url:, title:, icon_name:, fill_class:)
      @url = url
      @title = title
      @icon_name = icon_name
      @fill_class = fill_class
    end

    sig { returns(String) }
    attr_reader :url
    private :url

    sig { returns(String) }
    attr_reader :title
    private :title

    sig { returns(String) }
    attr_reader :icon_name
    private :icon_name

    sig { returns(String) }
    attr_reader :fill_class
    private :fill_class
  end
end
