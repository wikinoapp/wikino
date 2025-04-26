# typed: strict
# frozen_string_literal: true

class LinkListComponent < ApplicationComponent
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

  delegate :space, to: :page
  delegate :links, :pagination, to: :link_list
end
