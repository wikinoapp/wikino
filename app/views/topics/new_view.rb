# typed: strict
# frozen_string_literal: true

module Topics
  class NewView < ApplicationView
    sig do
      params(
        current_user: User,
        space: Space,
        form: NewTopicForm
      ).void
    end
    def initialize(current_user:, space:, form:)
      @current_user = current_user
      @space = space
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.topics.new", space_name: space.name)
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(User) }
    attr_reader :current_user
    private :current_user

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
