# typed: strict
# frozen_string_literal: true

class LinkCollectionComponent < ApplicationComponent
  sig { params(link_collection: LinkCollection).void }
  def initialize(link_collection:)
    @page = T.let(link_collection.page, Page)
    @links = T.let(link_collection.links, T::Array[Link])
    @pagination = T.let(link_collection.pagination, Pagination)
  end

  sig { returns(Page) }
  attr_reader :page
  private :page

  sig { returns(T::Array[Link]) }
  attr_reader :links
  private :links

  sig { returns(Pagination) }
  attr_reader :pagination
  private :pagination
end
