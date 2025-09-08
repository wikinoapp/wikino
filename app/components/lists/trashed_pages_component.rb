# typed: strict
# frozen_string_literal: true

module Lists
  class TrashedPagesComponent < ApplicationComponent
    sig { params(form: Pages::BulkRestoringForm, pages: T::Array[Page], space: Space).void }
    def initialize(form:, pages:, space:)
      @form = form
      @pages = pages
      @space = space
    end

    sig { returns(Pages::BulkRestoringForm) }
    attr_reader :form
    private :form

    sig { returns(T::Array[Page]) }
    attr_reader :pages
    private :pages

    sig { returns(Space) }
    attr_reader :space
    private :space
  end
end
