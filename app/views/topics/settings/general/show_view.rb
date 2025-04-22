# typed: strict
# frozen_string_literal: true

module Topics
  module Settings
    module General
      class ShowView < ApplicationView
        sig do
          params(
            current_user: User,
            topic_entity: TopicEntity,
            form: EditTopicForm
          ).void
        end
        def initialize(current_user:, topic_entity:, form:)
          @current_user = current_user
          @topic_entity = topic_entity
          @form = form
        end

        sig { override.void }
        def before_render
          title = I18n.t("meta.title.topics.settings.general.show",
            topic_name: topic_entity.name,
            space_name: space_entity.name)
          helpers.set_meta_tags(title:, **default_meta_tags(site: false))
        end

        sig { returns(User) }
        attr_reader :current_user
        private :current_user

        sig { returns(TopicEntity) }
        attr_reader :topic_entity
        private :topic_entity

        sig { returns(EditTopicForm) }
        attr_reader :form
        private :form

        delegate :space_entity, to: :topic_entity

        sig { returns(PageName) }
        private def current_page_name
          PageName::TopicSettingsGeneral
        end
      end
    end
  end
end
