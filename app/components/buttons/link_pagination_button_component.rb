# typed: strict
# frozen_string_literal: true

module Buttons
  class LinkPaginationButtonComponent < ApplicationComponent
    sig { params(link_collection: LinkCollection).void }
    def initialize(link_collection:)
      @link_collection = link_collection
    end

    sig { returns(LinkCollection) }
    attr_reader :link_collection
    private :link_collection

    delegate :page, :pagination, to: :link_collection
  end
end
