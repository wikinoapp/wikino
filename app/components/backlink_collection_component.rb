# typed: strict
# frozen_string_literal: true

class BacklinkCollectionComponent < ApplicationComponent
  sig { params(backlink_collection: BacklinkCollection).void }
  def initialize(backlink_collection:)
    @page = T.let(backlink_collection.page, Page)
    @backlinks = T.let(backlink_collection.backlinks, T::Array[Backlink])
    @pagination = T.let(backlink_collection.pagination, Pagination)
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
