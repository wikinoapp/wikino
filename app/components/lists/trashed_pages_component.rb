# typed: strict
# frozen_string_literal: true

module Lists
  class TrashedPagesComponent < ApplicationComponent
    sig { params(form: Pages::BulkRestoringForm, pages: T::Array[Page]).void }
    def initialize(form:, pages:)
      @form = form
      @pages = pages
    end

    sig { returns(Pages::BulkRestoringForm) }
    attr_reader :form
    private :form

    sig { returns(T::Array[Page]) }
    attr_reader :pages
    private :pages
  end
end
