# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowView < ApplicationView
    sig do
      params(
        space: Space,
        space_viewer: ModelConcerns::SpaceViewable,
        first_joined_topic: T.nilable(Topic),
        pinned_pages: Page::PrivateAssociationRelation,
        page_connection: PageConnection
      ).void
    end
    def initialize(space:, space_viewer:, first_joined_topic:, pinned_pages:, page_connection:)
      @space = space
      @space_viewer = space_viewer
      @first_joined_topic = first_joined_topic
      @pinned_pages = pinned_pages
      @page_connection = page_connection
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(ModelConcerns::SpaceViewable) }
    attr_reader :space_viewer
    private :space_viewer

    sig { returns(T.nilable(Topic)) }
    attr_reader :first_joined_topic
    private :first_joined_topic

    sig { returns(Page::PrivateAssociationRelation) }
    attr_reader :pinned_pages
    private :pinned_pages

    sig { returns(PageConnection) }
    attr_reader :page_connection
    private :page_connection

    delegate :pages, :pagination, to: :page_connection
  end
end
