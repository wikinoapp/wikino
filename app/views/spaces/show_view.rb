# typed: strict
# frozen_string_literal: true

module Spaces
  class ShowView < ApplicationView
    sig do
      params(
        signed_in: T::Boolean,
        space_entity: SpaceEntity,
        first_topic_entity: T.nilable(TopicEntity),
        pinned_page_entities: T::Array[PageEntity],
        page_list_entity: PageListEntity
      ).void
    end
    def initialize(signed_in:, space_entity:, first_topic_entity:, pinned_page_entities:, page_list_entity:)
      @signed_in = signed_in
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

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in
    alias_method :signed_in?, :signed_in

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

    sig { returns(PageName) }
    private def current_page_name
      PageName::SpaceDetail
    end
  end
end
