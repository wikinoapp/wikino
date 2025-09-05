# typed: strict
# frozen_string_literal: true

module CardLinks
  class TopicCardComponent < ApplicationComponent
    sig { params(topic: Topic, current_user_record: T.nilable(UserRecord), card_class: String).void }
    def initialize(topic:, current_user_record:, card_class: "")
      @topic = topic
      @current_user_record = current_user_record
      @card_class = card_class
    end

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(T.nilable(UserRecord)) }
    attr_reader :current_user_record
    private :current_user_record

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
      return false if current_user_record.nil?

      topic.can_create_page?
    end

    sig { returns(T::Boolean) }
    private def can_update?
      return false if current_user_record.nil?

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
