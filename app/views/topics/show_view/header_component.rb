# typed: strict
# frozen_string_literal: true

module Topics
  class ShowView
    class HeaderComponent < ApplicationComponent
      sig { params(topic_entity: TopicEntity).void }
      def initialize(topic_entity:)
        @topic_entity = topic_entity
      end

      sig { returns(TopicEntity) }
      attr_reader :topic_entity
      private :topic_entity

      delegate :space_entity, to: :topic_entity
    end
  end
end
