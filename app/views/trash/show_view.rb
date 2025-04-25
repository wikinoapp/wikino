# typed: strict
# frozen_string_literal: true

module Trash
  class ShowView < ApplicationView
    sig do
      params(
        current_user: User,
        space: Space,
        page_list: PageList,
        form: TrashedPagesForm
      ).void
    end
    def initialize(current_user:, space:, page_list:, form:)
      @current_user = current_user
      @space = space
      @page_list = page_list
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.trash.show", space_name: space.name)
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(User) }
    attr_reader :current_user
    private :current_user

    sig { returns(Space) }
    attr_reader :space
    private :space

    sig { returns(PageList) }
    attr_reader :page_list
    private :page_list

    sig { returns(TrashedPagesForm) }
    attr_reader :form
    private :form

    delegate :page_entities, :pagination, to: :page_list

    sig { returns(PageName) }
    private def current_page_name
      PageName::Trash
    end
  end
end
