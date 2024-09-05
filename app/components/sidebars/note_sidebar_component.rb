# typed: strict
# frozen_string_literal: true

module Sidebars
  class NoteSidebarComponent < ApplicationComponent
    sig { params(links: T::Array[Link], backlinks: T::Array[Backlink]).void }
    def initialize(links:, backlinks:)
      @links = links
      @backlinks = backlinks
    end

    sig { returns(T::Array[Link]) }
    attr_reader :links
    private :links

    sig { returns(T::Array[Backlink]) }
    attr_reader :backlinks
    private :backlinks
  end
end
