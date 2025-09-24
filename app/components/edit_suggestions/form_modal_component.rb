# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class FormModalComponent < ApplicationComponent
    sig do
      params(
        page: Page,
        existing_edit_suggestions: T::Array[EditSuggestion]
      ).void
    end
    def initialize(page:, existing_edit_suggestions:)
      @page = page
      @existing_edit_suggestions = existing_edit_suggestions
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(T::Array[EditSuggestion]) }
    attr_reader :existing_edit_suggestions
    private :existing_edit_suggestions

    delegate :space, to: :page, private: true
    delegate :topic, to: :page, private: true

    sig { returns(T.nilable(String)) }
    private def add_to_existing_path
      helpers.new_edit_suggestion_page_path(
        space_identifier: space.identifier,
        topic_number: topic.number,
        page_number: page.not_nil!.number
      )
    end

    sig { returns(T::Boolean) }
    private def has_existing_edit_suggestions?
      existing_edit_suggestions.any?
    end
  end
end
