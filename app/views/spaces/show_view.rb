# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowView < ApplicationView
    use_helpers :set_meta_tags

    sig do
      params(
        space_viewer: ModelConcerns::SpaceViewable,
        pinned_pages: Page::PrivateAssociationRelation,
        page_connection: PageConnection
      ).void
    end
    def initialize(space_viewer:, pinned_pages:, page_connection:)
      @space_viewer = space_viewer
      @pinned_pages = pinned_pages
      @page_connection = page_connection
    end

    sig { returns(ModelConcerns::SpaceViewable) }
    attr_reader :space_viewer
    private :space_viewer

    sig { returns(Page::PrivateAssociationRelation) }
    attr_reader :pinned_pages
    private :pinned_pages

    sig { returns(PageConnection) }
    attr_reader :page_connection
    private :page_connection

    delegate :pages, :pagination, to: :page_connection
    delegate :space, to: :space_viewer

    sig { returns(T.nilable(Topic)) }
    private def first_joined_topic
      space_viewer.joined_topics.first
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::SpaceDetail
    end
  end
end
