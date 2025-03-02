# typed: strict
# frozen_string_literal: true

module Lists
  class TrashedPagesComponent < ApplicationComponent
    sig { params(form: TrashedPagesForm, page_entities: T::Array[PageEntity]).void }
    def initialize(form:, page_entities:)
      @form = form
      @page_entities = page_entities
    end

    sig { returns(TrashedPagesForm) }
    attr_reader :form
    private :form

    sig { returns(T::Array[PageEntity]) }
    attr_reader :page_entities
    private :page_entities
  end
end
