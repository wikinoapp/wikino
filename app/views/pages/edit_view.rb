# typed: strict
# frozen_string_literal: true

module Pages
  class EditView < ApplicationView
    sig do
      params(
        space_entity: SpaceEntity,
        page_entity: PageEntity,
        form: EditPageForm,
        link_list_entity: LinkListEntity,
        backlink_list_entity: BacklinkListEntity,
        draft_page_entity: T.nilable(DraftPageEntity)
      ).void
    end
    def initialize(space_entity:, page_entity:, form:, link_list_entity:, backlink_list_entity:, draft_page_entity: nil)
      @space_entity = space_entity
      @page_entity = page_entity
      @form = form
      @link_list_entity = link_list_entity
      @backlink_list_entity = backlink_list_entity
      @draft_page_entity = draft_page_entity
    end

    sig { override.void }
    def before_render
      helpers.set_meta_tags(title: "#{title} | #{space_entity.name}", **default_meta_tags)
    end

    sig { returns(SpaceEntity) }
    attr_reader :space_entity
    private :space_entity

    sig { returns(PageEntity) }
    attr_reader :page_entity
    private :page_entity

    sig { returns(EditPageForm) }
    attr_reader :form
    private :form

    sig { returns(LinkListEntity) }
    attr_reader :link_list_entity
    private :link_list_entity

    sig { returns(BacklinkListEntity) }
    attr_reader :backlink_list_entity
    private :backlink_list_entity

    sig { returns(T.nilable(DraftPageEntity)) }
    attr_reader :draft_page_entity
    private :draft_page_entity

    sig { returns(String) }
    private def title
      I18n.t("meta.title.pages.edit")
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::PageEdit
    end
  end
end
