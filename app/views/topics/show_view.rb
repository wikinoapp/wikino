# typed: strict
# frozen_string_literal: true

module Topics
  class ShowView < ApplicationView
    sig do
      params(
        signed_in: T::Boolean,
        topic_entity: TopicEntity,
        pinned_page_entities: T::Array[PageEntity],
        page_list_entity: PageListEntity
      ).void
    end
    def initialize(signed_in:, topic_entity:, pinned_page_entities:, page_list_entity:)
      @signed_in = signed_in
      @topic_entity = topic_entity
      @pinned_page_entities = pinned_page_entities
      @page_list_entity = page_list_entity
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.topics.show", space_name: space_entity.name, topic_name: topic_entity.name)
      helpers.set_meta_tags(title:, **default_meta_tags(site: false))
    end

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in
    alias_method :signed_in?, :signed_in

    sig { returns(TopicEntity) }
    attr_reader :topic_entity
    private :topic_entity

    sig { returns(T::Array[PageEntity]) }
    attr_reader :pinned_page_entities
    private :pinned_page_entities

    sig { returns(PageListEntity) }
    attr_reader :page_list_entity
    private :page_list_entity

    delegate :space_entity, to: :topic_entity
    delegate :page_entities, :pagination_entity, to: :page_list_entity

    sig { returns(PageName) }
    private def current_page_name
      PageName::TopicDetail
    end
  end
end
