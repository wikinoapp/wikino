# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class FormModalComponent < ApplicationComponent
    sig do
      params(
        space: Space,
        topic: Topic,
        page: T.nilable(Page),
        page_title: String,
        page_body: String,
        existing_edit_suggestions: T::Array[EditSuggestion]
      ).void
    end
    def initialize(space:, topic:, page:, page_title:, page_body:, existing_edit_suggestions:)
      @space = space
      @topic = topic
      @page = page
      @page_title = page_title
      @page_body = page_body
      @existing_edit_suggestions = existing_edit_suggestions
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    sig { returns(T.nilable(Page)) }
    attr_reader :page
    private :page

    sig { returns(String) }
    attr_reader :page_title
    private :page_title

    sig { returns(String) }
    attr_reader :page_body
    private :page_body

    sig { returns(T::Array[EditSuggestion]) }
    attr_reader :existing_edit_suggestions
    private :existing_edit_suggestions

    sig { returns(String) }
    private def new_edit_suggestion_form_path
      if page
        helpers.new_edit_suggestion_path(
          space_identifier: space.identifier,
          topic_number: topic.number,
          page_number: page&.number,
          page_title:,
          page_body:
        )
      else
        helpers.new_edit_suggestion_path(
          space_identifier: space.identifier,
          topic_number: topic.number,
          page_title:,
          page_body:
        )
      end
    end

    sig { params(edit_suggestion: EditSuggestion).returns(String) }
    private def add_to_existing_path(edit_suggestion)
      if page
        helpers.new_edit_suggestion_page_path(
          space_identifier: space.identifier,
          topic_number: topic.number,
          id: edit_suggestion.database_id,
          page_number: page&.number,
          page_title:,
          page_body:
        )
      else
        helpers.new_edit_suggestion_page_path(
          space_identifier: space.identifier,
          topic_number: topic.number,
          id: edit_suggestion.database_id,
          page_title:,
          page_body:
        )
      end
    end

    sig { returns(T::Boolean) }
    private def has_existing_edit_suggestions?
      existing_edit_suggestions.any?
    end
  end
end
