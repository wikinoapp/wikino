# typed: strict
# frozen_string_literal: true

module Trash
  class ShowView < ApplicationView
    sig { params(space: Space, page_connection: PageConnection, form: TrashedPagesForm).void }
    def initialize(space:, page_connection:, form:)
      @space = space
      @page_connection = page_connection
      @form = form
      @current_page_name = PageName::Trash
    end

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(PageConnection) }
    attr_reader :page_connection
    private :page_connection

    sig { returns(TrashedPagesForm) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    attr_reader :current_page_name
    private :current_page_name

    delegate :pages, :pagination, to: :page_connection
  end
end
