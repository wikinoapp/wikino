# typed: strict
# frozen_string_literal: true

class PageLinksComponent < ApplicationComponent
  sig { params(page: Page, link_list: LinkList, backlink_list: BacklinkList).void }
  def initialize(page:, link_list:, backlink_list:)
    @page = page
    @link_list = link_list
    @backlink_list = backlink_list
  end

  sig { returns(Page) }
  attr_reader :page
  private :page

  sig { returns(LinkList) }
  attr_reader :link_list
  private :link_list

  sig { returns(BacklinkList) }
  attr_reader :backlink_list
  private :backlink_list
end
