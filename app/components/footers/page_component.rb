# typed: strict
# frozen_string_literal: true

module Footers
  class PageComponent < ApplicationComponent
    sig { params(link_collection: ::LinkCollection, backlink_collection: ::BacklinkCollection).void }
    def initialize(link_collection:, backlink_collection:)
      @link_collection = link_collection
      @backlink_collection = backlink_collection
    end

    sig { returns(::LinkCollection) }
    attr_reader :link_collection
    private :link_collection

    sig { returns(::BacklinkCollection) }
    attr_reader :backlink_collection
    private :backlink_collection

    sig { returns(T::Boolean) }
    def render?
      show_links? || show_backlinks?
    end

    sig { returns(T::Boolean) }
    private def show_links?
      link_collection.links.present?
    end

    sig { returns(T::Boolean) }
    private def show_backlinks?
      backlink_collection.backlinks.present?
    end
  end
end
