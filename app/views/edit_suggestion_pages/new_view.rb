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
        edit_suggestion: EditSuggestion
      ).void
    end
    def initialize(form:, space:, topic:, page:, edit_suggestion:)
      @form = form
      @space = space
      @topic = topic
      @page = page
      @edit_suggestion = edit_suggestion
    end

    sig { returns(EditSuggestionPages::CreateForm) }
    attr_reader :form

    sig { returns(Space) }
    attr_reader :space

    sig { returns(Topic) }
    attr_reader :topic

    sig { returns(T.nilable(Page)) }
    attr_reader :page

    sig { returns(EditSuggestion) }
    attr_reader :edit_suggestion

    sig { returns(String) }
    def form_action_path
      if page
        edit_suggestion_page_list_path(
          space_identifier: space.identifier,
          topic_number: topic.number,
          id: edit_suggestion.database_id,
          page_number: page.not_nil!.number
        )
      else
        edit_suggestion_page_list_path(
          space_identifier: space.identifier,
          topic_number: topic.number,
          id: edit_suggestion.database_id
        )
      end
    end
  end
end
