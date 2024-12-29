# typed: strict
# frozen_string_literal: true

module Cards
  class TopicCardComponent < ApplicationComponent
    sig { params(topic: Topic, class_name: String).void }
    def initialize(topic:, class_name: "")
      @topic = topic
      @class_name = class_name
    end

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
