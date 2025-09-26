# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class NewView < ApplicationView
    sig do
      params(
        form: EditSuggestionPages::CreateForm,
        page: Page,
        existing_edit_suggestions: T::Array[EditSuggestion]
      ).void
    end
    def initialize(form:, page:, existing_edit_suggestions:)
      @form = form
      @page = page
      @existing_edit_suggestions = existing_edit_suggestions
    end

    sig { returns(EditSuggestionPages::CreateForm) }
    attr_reader :form
    private :form

    sig { returns(Page) }
    attr_reader :page
    private :page

    sig { returns(T::Array[EditSuggestion]) }
    attr_reader :existing_edit_suggestions
    private :existing_edit_suggestions

    delegate :space, to: :page, private: true
    delegate :topic, to: :page, private: true
  end
end
