# typed: strict
# frozen_string_literal: true

module Topics
  class ShowView < ApplicationView
    use_helpers :set_meta_tags

    sig do
      params(
        topic: Topic,
        space_viewer: ModelConcerns::SpaceViewable,
        pinned_pages: Page::PrivateRelation,
        page_connection: PageConnection
      ).void
    end
    def initialize(topic:, space_viewer:, pinned_pages:, page_connection:)
      @topic = topic
      @space_viewer = space_viewer
      @pinned_pages = pinned_pages
      @page_connection = page_connection
    end

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(ModelConcerns::SpaceViewable) }
    attr_reader :space_viewer
    private :space_viewer

    sig { returns(Page::PrivateRelation) }
    attr_reader :pinned_pages
    private :pinned_pages

    sig { returns(PageConnection) }
    attr_reader :page_connection
    private :page_connection

    delegate :space, to: :topic
    delegate :pages, :pagination, to: :page_connection
  end
end
