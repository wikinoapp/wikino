# typed: strict
# frozen_string_literal: true

module Topics
  module Settings
    class ShowView < ApplicationView
      sig { params(topic_entity: TopicEntity).void }
      def initialize(topic_entity:)
        @topic_entity = topic_entity
      end

      sig { override.void }
      def before_render
        title = I18n.t("meta.title.topics.settings.show", topic_name: topic_entity.name, space_name: space_entity.name)
        helpers.set_meta_tags(title:, **default_meta_tags(site: false))
      end

      sig { returns(TopicEntity) }
      attr_reader :topic_entity
      private :topic_entity

      delegate :space_entity, to: :topic_entity

      sig { returns(PageName) }
      private def current_page_name
        PageName::TopicSettings
      end
    end
  end
end
