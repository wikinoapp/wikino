# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class NewView < ApplicationView
    sig do
      params(
        form: EditSuggestions::CreateForm,
        page: Page
      ).void
    end
    def initialize(form:, page:)
      @form = form
      @page = page
    end

    sig { returns(EditSuggestions::CreateForm) }
    attr_reader :form
    private :form

    sig { returns(Page) }
    attr_reader :page
    private :page

    delegate :space, to: :page, private: true
    delegate :topic, to: :page, private: true

    sig { returns(String) }
    private def form_action_path
      edit_suggestion_list_path(
        space_identifier: space.identifier,
        page_number: page.not_nil!.number
      )
    end
  end
end
