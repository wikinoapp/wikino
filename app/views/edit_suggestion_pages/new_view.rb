# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class NewView < ApplicationView
    sig do
      params(
        form: EditSuggestionPages::CreateForm,
        space: Space,
        topic: Topic,
        page: T.nilable(Page),
        existing_edit_suggestions: T::Array[EditSuggestion]
      ).void
    end
    def initialize(form:, space:, topic:, page:, existing_edit_suggestions:)
      @form = form
      @space = space
      @topic = topic
      @page = page
      @existing_edit_suggestions = existing_edit_suggestions
    end

    sig { returns(EditSuggestionPages::CreateForm) }
    attr_reader :form

    sig { returns(Space) }
    attr_reader :space

    sig { returns(Topic) }
    attr_reader :topic

    sig { returns(T.nilable(Page)) }
    attr_reader :page

    sig { returns(T::Array[EditSuggestion]) }
    attr_reader :existing_edit_suggestions

    sig { returns(String) }
    def form_action_path
      if page
        edit_suggestion_page_list_path(
          space_identifier: space.identifier,
          topic_number: topic.number,
          page_number: page.not_nil!.number
        )
      else
        edit_suggestion_page_list_path(
          space_identifier: space.identifier,
          topic_number: topic.number
        )
      end
    end
  end
end
