# typed: strict
# frozen_string_literal: true

class BacklinkCollectionComponent < ApplicationComponent
  sig { params(backlink_collection: BacklinkCollection).void }
  def initialize(backlink_collection:)
    @page = backlink_collection.page
    @backlinks = backlink_collection.backlinks
    @pagination = backlink_collection.pagination
  end

  sig { returns(Page) }
  attr_reader :page
  private :page

  sig { returns(T::Array[Backlink]) }
  attr_reader :backlinks
  private :backlinks

  sig { returns(Pagination) }
  attr_reader :pagination
  private :pagination
end
