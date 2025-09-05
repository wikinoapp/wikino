# typed: strict
# frozen_string_literal: true

module CardLinks
  class TopicComponent < ApplicationComponent
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

    sig { returns(String) }
    private def build_card_class
      class_names(
        card_class,
        "bg-card duration-200 ease-in-out grid min-h-[80px] transition px-3 py-2",
        "hover:border hover:border-primary"
      )
    end

    sig { returns(T::Boolean) }
    private def can_create_page?
      topic.can_create_page?
    end

    sig { returns(T::Boolean) }
    private def can_update?
      topic.can_update?
    end

    sig { returns(String) }
    private def topic_path
      Rails.application.routes.url_helpers.topic_path(
        space_identifier: topic.space.identifier,
        topic_number: topic.number
      )
    end

    sig { returns(String) }
    private def new_page_path
      Rails.application.routes.url_helpers.new_page_path(
        space_identifier: topic.space.identifier,
        topic_number: topic.number
      )
    end

    sig { returns(String) }
    private def settings_path
      Rails.application.routes.url_helpers.topic_settings_path(
        space_identifier: topic.space.identifier,
        topic_number: topic.number
      )
    end
  end
end
