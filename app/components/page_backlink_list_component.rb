# typed: strict
# frozen_string_literal: true

class PageBacklinkListComponent < ApplicationComponent
  sig { params(backlink_list: BacklinkList).void }
  def initialize(backlink_list:)
    @backlink_list = backlink_list
  end

  sig { returns(BacklinkList) }
  attr_reader :backlink_list
  private :backlink_list
end
