# typed: strict
# frozen_string_literal: true

class LinkCollectionComponent < ApplicationComponent
  sig { params(link_collection: LinkCollection).void }
  def initialize(link_collection:)
    @page = link_collection.page
    @links = link_collection.links
    @pagination = link_collection.pagination
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
