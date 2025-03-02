# typed: strict
# frozen_string_literal: true

module Trash
  class ShowView < ApplicationView
    sig { params(space_entity: SpaceEntity, page_list_entity: PageListEntity, form: TrashedPagesForm).void }
    def initialize(space_entity:, page_list_entity:, form:)
      @space_entity = space_entity
      @page_list_entity = page_list_entity
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.trash.show", space_name: space_entity.name)
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(SpaceEntity) }
    attr_reader :space_entity
    private :space_entity

    sig { returns(PageListEntity) }
    attr_reader :page_list_entity
    private :page_list_entity

    sig { returns(TrashedPagesForm) }
    attr_reader :form
    private :form

    delegate :page_entities, :pagination_entity, to: :page_list_entity

    sig { returns(PageName) }
    private def current_page_name
      PageName::Trash
    end
  end
end
