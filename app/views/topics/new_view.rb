# typed: strict
# frozen_string_literal: true

module Topics
  class NewView < ApplicationView
    use_helpers :set_meta_tags

    sig { params(space: Space, form: NewTopicForm).void }
    def initialize(space:, form:)
      @space = space
      @form = form
      @current_page_name = PageName::TopicNew
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(NewTopicForm) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name
  end
end
