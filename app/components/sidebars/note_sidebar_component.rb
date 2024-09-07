# typed: strict
# frozen_string_literal: true

module Sidebars
  class NoteSidebarComponent < ApplicationComponent
    sig { params(link_list: LinkList, backlinks: T::Array[Backlink]).void }
    def initialize(link_list:, backlinks:)
      @link_list = link_list
      @backlinks = backlinks
    end

    sig { returns(LinkList) }
    attr_reader :link_list
    private :link_list

    sig { returns(T::Array[Backlink]) }
    attr_reader :backlinks
    private :backlinks
  end
end
