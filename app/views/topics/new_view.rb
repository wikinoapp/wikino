# typed: strict
# frozen_string_literal: true

module Topics
  class NewView < ApplicationView
    sig { params(space: Space, form: NewTopicForm).void }
    def initialize(space:, form:)
      @space = space
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.topics.new")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(NewTopicForm) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::TopicNew
    end
  end
end
