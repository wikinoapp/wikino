# typed: strict
# frozen_string_literal: true

module CardLinks
  class TopicWithActionComponent < ApplicationComponent
    sig { params(topic: Topic, card_class: String).void }
    def initialize(topic:, card_class: "")
      @topic = topic
      @card_class = card_class
    end

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(String) }
    attr_reader :card_class
    private :card_class

    sig { returns(T::Boolean) }
    private def can_create_page?
      topic.can_create_page?
    end

    sig { returns(String) }
    private def new_page_path
      Rails.application.routes.url_helpers.new_page_path(
        space_identifier: topic.space.identifier,
        topic_number: topic.number
      )
    end
  end
end
