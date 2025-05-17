# typed: strict
# frozen_string_literal: true

module Breadcrumbs
  class TopicBreadcrumbsComponent < ApplicationComponent
    sig { params(topic: Topic).void }
    def initialize(topic:)
      @topic = topic
    end

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    delegate :space, to: :topic
  end
end
