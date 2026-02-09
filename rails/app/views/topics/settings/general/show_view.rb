# typed: strict
# frozen_string_literal: true

module Topics
  module Settings
    module General
      class ShowView < ApplicationView
        sig do
          params(
            current_user: User,
            topic: Topic,
            form: Topics::EditForm
          ).void
        end
        def initialize(current_user:, topic:, form:)
          @current_user = current_user
          @topic = topic
          @form = form
        end

        sig { override.void }
        def before_render
          title = I18n.t("meta.title.topics.settings.general.show",
            topic_name: topic.name,
            space_name: space.name)
          helpers.set_meta_tags(title:, **default_meta_tags(site: false))
        end

        sig { returns(User) }
        attr_reader :current_user
        private :current_user

        sig { returns(Topic) }
        attr_reader :topic
        private :topic

        sig { returns(Topics::EditForm) }
        attr_reader :form
        private :form

        delegate :space, to: :topic

        sig { returns(PageName) }
        private def current_page_name
          PageName::TopicSettingsGeneral
        end
      end
    end
  end
end
