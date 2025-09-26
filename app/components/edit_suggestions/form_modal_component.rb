# typed: strict
# frozen_string_literal: true

module EditSuggestions
  class FormModalComponent < ApplicationComponent
    sig { params(page: Page).void }
    def initialize(page:)
      @page = page
    end

    sig { returns(Page) }
    attr_reader :page
    private :page

    delegate :space, to: :page, private: true
    delegate :topic, to: :page, private: true
  end
end
