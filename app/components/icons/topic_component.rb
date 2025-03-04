# typed: strict
# frozen_string_literal: true

module Icons
  class TopicComponent < ApplicationComponent
    sig { params(topic_entity: TopicEntity, size: String, class_name: String).void }
    def initialize(topic_entity:, size: "16px", class_name: "")
      @topic_entity = topic_entity
      @size = size
      @class_name = class_name
    end

    sig { returns(::TopicEntity) }
    attr_reader :topic_entity
    private :topic_entity

    sig { returns(String) }
    attr_reader :size
    private :size

    sig { returns(String) }
    attr_reader :class_name
    private :class_name

    sig { returns(String) }
    private def icon_name
      topic_entity.visibility_public? ? "globe" : "lock"
    end
  end
end
