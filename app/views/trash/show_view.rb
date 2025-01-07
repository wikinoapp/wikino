# typed: strict
# frozen_string_literal: true

module Trash
  class ShowView < ApplicationView
    sig { params(page_connection: PageConnection, form: TrashedPagesForm).void }
    def initialize(page_connection:, form:)
      @page_connection = page_connection
      @form = form
    end

    sig { returns(PageConnection) }
    attr_reader :page_connection
    private :page_connection

    sig { returns(TrashedPagesForm) }
    attr_reader :form
    private :form

    delegate :pages, :pagination, to: :page_connection
  end
end
