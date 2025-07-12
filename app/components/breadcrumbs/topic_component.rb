# typed: strict
# frozen_string_literal: true

module Breadcrumbs
  class TopicComponent < ApplicationComponent
    renders_many :items, BaseUI::BreadcrumbComponent::Item

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
