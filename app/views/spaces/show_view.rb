# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowView < ApplicationView
    sig do
      params(
        current_user_entity: T.nilable(UserEntity),
        space_entity: SpaceEntity,
        first_topic_entity: T.nilable(TopicEntity),
        pinned_page_entities: T::Array[PageEntity],
        page_list_entity: PageListEntity
      ).void
    end
    def initialize(current_user_entity:, space_entity:, first_topic_entity:, pinned_page_entities:, page_list_entity:)
      @current_user_entity = current_user_entity
      @space_entity = space_entity
      @first_topic_entity = first_topic_entity
      @pinned_page_entities = pinned_page_entities
      @page_list_entity = page_list_entity
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.spaces.show", space_name: space_entity.name)
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(T.nilable(UserEntity)) }
    attr_reader :current_user_entity
    private :current_user_entity

    sig { returns(SpaceEntity) }
    attr_reader :space_entity
    private :space_entity

    sig { returns(T.nilable(TopicEntity)) }
    attr_reader :first_topic_entity
    private :first_topic_entity

    sig { returns(T::Array[PageEntity]) }
    attr_reader :pinned_page_entities
    private :pinned_page_entities

    sig { returns(PageListEntity) }
    attr_reader :page_list_entity
    private :page_list_entity

    delegate :page_entities, :pagination_entity, to: :page_list_entity

    sig { returns(T::Boolean) }
    private def signed_in?
      !current_user_entity.nil?
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::SpaceDetail
    end
  end
end
