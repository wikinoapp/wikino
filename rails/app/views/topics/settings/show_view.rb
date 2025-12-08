# typed: strict
# frozen_string_literal: true

module Topics
  module Settings
    class ShowView < ApplicationView
      sig { params(current_user: User, topic: Topic).void }
      def initialize(current_user:, topic:)
        @current_user = current_user
        @topic = topic
      end

      sig { override.void }
      def before_render
        title = I18n.t("meta.title.topics.settings.show", topic_name: topic.name, space_name: space.name)
        helpers.set_meta_tags(title:, **default_meta_tags(site: false))
      end

      sig { returns(User) }
      attr_reader :current_user
      private :current_user

      sig { returns(Topic) }
      attr_reader :topic
      private :topic

      delegate :space, to: :topic

      sig { returns(PageName) }
      private def current_page_name
        PageName::TopicSettings
      end
    end
  end
end
