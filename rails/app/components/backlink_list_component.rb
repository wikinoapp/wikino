# typed: strict
# frozen_string_literal: true

class BacklinkListComponent < ApplicationComponent
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

  delegate :space, to: :page
  delegate :backlinks, :pagination, to: :backlink_list
end
