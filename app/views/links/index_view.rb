# typed: strict
# frozen_string_literal: true

module Links
  class IndexView < ApplicationView
    sig { params(page: Page, link_list: LinkList).void }
    def initialize(page:, link_list:)
      @page = page
      @link_list = link_list
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(LinkList) }
    attr_reader :link_list
    private :link_list
  end
end
