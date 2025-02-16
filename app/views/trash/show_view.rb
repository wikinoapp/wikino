# typed: strict
# frozen_string_literal: true

module Trash
  class ShowView < ApplicationView
    sig { params(space: Space, page_connection: PageConnection, form: TrashedPagesForm).void }
    def initialize(space:, page_connection:, form:)
      @space = space
      @page_connection = page_connection
      @form = form
    end

    def before_render
      title = I18n.t("meta.title.trash.show", space_name: space.name)
      helpers.set_meta_tags(title:, **default_meta_tags)
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

    delegate :pages, :pagination, to: :page_connection

    sig { returns(PageName) }
    private def current_page_name
      PageName::Trash
    end
  end
end
