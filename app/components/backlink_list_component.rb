# typed: strict
# frozen_string_literal: true

class BacklinkListComponent < ApplicationComponent
  sig { params(page: Page, backlink_list: BacklinkList, target: String).void }
  def initialize(page:, backlink_list:, target: "_self")
    @page = page
    @backlink_list = backlink_list
    @target = target
  end

  sig { returns(Page) }
  attr_reader :page
  private :page

  sig { returns(BacklinkList) }
  attr_reader :backlink_list
  private :backlink_list

  sig { returns(String) }
  attr_reader :target
  private :target

  delegate :space, to: :page
  delegate :backlinks, :pagination, to: :backlink_list
end
