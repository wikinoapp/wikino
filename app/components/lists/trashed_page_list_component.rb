# typed: strict
# frozen_string_literal: true

module Lists
  class TrashedPageListComponent < ApplicationComponent
    sig { params(form: TrashedPagesForm, pages: T::Array[Page]).void }
    def initialize(form:, pages:)
      @form = form
      @pages = pages
    end

    sig { returns(TrashedPagesForm) }
    attr_reader :form
    private :form

    sig { returns(T::Array[Page]) }
    attr_reader :pages
    private :pages
  end
end
