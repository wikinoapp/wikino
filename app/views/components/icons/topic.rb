# typed: strict
# frozen_string_literal: true

module Views
  module Components
    module Icons
      class Topic < Views::Components::Base
        sig { params(topic: ::Topic, size: String, class_name: String).void }
        def initialize(topic:, size: "16px", class_name: "")
          @topic = topic
          @size = size
          @class_name = class_name
        end

        sig { returns(::Topic) }
        attr_reader :topic
        private :topic

        sig { returns(String) }
        attr_reader :size
        private :size

        sig { returns(String) }
        attr_reader :class_name
        private :class_name

        sig { returns(String) }
        private def icon_name
          topic.visibility_public? ? "globe" : "lock"
        end
      end
    end
  end
end
