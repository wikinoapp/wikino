# typed: strict
# frozen_string_literal: true

module Backlinks
  class IndexView < ApplicationView
    sig { params(page: Page, backlink_list: BacklinkList).void }
    def initialize(page:, backlink_list:)
      @page = page
      @backlink_list = backlink_list
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(BacklinkList) }
    attr_reader :backlink_list
    private :backlink_list
  end
end
