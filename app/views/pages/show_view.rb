# typed: strict
# frozen_string_literal: true

module Pages
  class ShowView < ApplicationView
    sig do
      params(
        current_user_entity: T.nilable(UserEntity),
        page_entity: PageEntity,
        link_list_entity: LinkListEntity,
        backlink_list_entity: BacklinkListEntity
      ).void
    end
    def initialize(current_user_entity:, page_entity:, link_list_entity:, backlink_list_entity:)
      @current_user_entity = current_user_entity
      @page_entity = page_entity
      @link_list_entity = link_list_entity
      @backlink_list_entity = backlink_list_entity
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.pages.show", space_name: space_entity.name, page_title: page_entity.title)
      helpers.set_meta_tags(title:, **default_meta_tags(site: false))
    end

    sig { returns(T.nilable(UserEntity)) }
    attr_reader :current_user_entity
    private :current_user_entity

    sig { returns(PageEntity) }
    attr_reader :page_entity
    private :page_entity

    sig { returns(LinkListEntity) }
    attr_reader :link_list_entity
    private :link_list_entity

    sig { returns(BacklinkListEntity) }
    attr_reader :backlink_list_entity
    private :backlink_list_entity

    delegate :space_entity, :topic_entity, to: :page_entity

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user_entity.nil?
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::PageDetail
    end
  end
end
