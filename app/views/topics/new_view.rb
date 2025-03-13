# typed: strict
# frozen_string_literal: true

module Topics
  class NewView < ApplicationView
    sig { params(space_entity: SpaceEntity, form: NewTopicForm).void }
    def initialize(space_entity:, form:)
      @space_entity = space_entity
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.topics.new", space_name: space_entity.name)
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(SpaceEntity) }
    attr_reader :space_entity
    private :space_entity

    sig { returns(NewTopicForm) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::TopicNew
    end
  end
end
