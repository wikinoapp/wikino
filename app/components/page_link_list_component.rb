# typed: strict
# frozen_string_literal: true

class PageLinkListComponent < ApplicationComponent
  sig { params(link_list: LinkList).void }
  def initialize(link_list:)
    @link_list = link_list
  end

  sig { returns(LinkList) }
  attr_reader :link_list
  private :link_list
end
