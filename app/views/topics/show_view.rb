# typed: strict
# frozen_string_literal: true

module Topics
  class ShowView < ApplicationView
    use_helpers :policy

    sig { params(topic: Topic, pinned_pages: Page::PrivateRelation, page_connection: PageConnection).void }
    def initialize(topic:, pinned_pages:, page_connection:)
      @topic = topic
      @pinned_pages = pinned_pages
      @page_connection = page_connection
    end

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(Page::PrivateRelation) }
    attr_reader :pinned_pages
    private :pinned_pages

    sig { returns(PageConnection) }
    attr_reader :page_connection
    private :page_connection

    delegate :pages, :pagination, to: :page_connection
  end
end
