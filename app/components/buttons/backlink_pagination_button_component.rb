# typed: strict
# frozen_string_literal: true

module Buttons
  class BacklinkPaginationButtonComponent < ApplicationComponent
    sig { params(backlink_collection: BacklinkCollection).void }
    def initialize(backlink_collection:)
      @backlink_collection = backlink_collection
    end

    sig { returns(BacklinkCollection) }
    attr_reader :backlink_collection
    private :backlink_collection

    delegate :page, :pagination, to: :backlink_collection
  end
end
