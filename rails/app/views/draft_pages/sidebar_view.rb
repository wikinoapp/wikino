# typed: strict
# frozen_string_literal: true

module DraftPages
  class SidebarView < ApplicationView
    sig { params(draft_pages: T::Array[DraftPage], has_more: T::Boolean).void }
    def initialize(draft_pages:, has_more:)
      @draft_pages = draft_pages
      @has_more = has_more
    end

    sig { returns(T::Array[DraftPage]) }
    attr_reader :draft_pages
    private :draft_pages

    sig { returns(T::Boolean) }
    attr_reader :has_more
    private :has_more

    sig { params(draft_page: DraftPage).returns(String) }
    private def display_title(draft_page)
      draft_page.display_title
    end
  end
end
