# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowView < ApplicationView
    use_helpers :signed_in?

    sig { params(pinned_pages: Page::PrivateAssociationRelation, page_connection: PageConnection).void }
    def initialize(pinned_pages:, page_connection:)
      @pinned_pages = pinned_pages
      @page_connection = page_connection
    end

    sig { returns(Page::PrivateAssociationRelation) }
    attr_reader :pinned_pages
    private :pinned_pages

    sig { returns(PageConnection) }
    attr_reader :page_connection
    private :page_connection

    delegate :pages, :pagination, to: :page_connection
  end
end
