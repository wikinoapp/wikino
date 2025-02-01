# typed: strict
# frozen_string_literal: true

module Topics
  class NewView < ApplicationView
    def initialize(space:, form:)
      @space = space
      @form = form
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(NewTopicForm) }
    attr_reader :form
    private :form
  end
end
