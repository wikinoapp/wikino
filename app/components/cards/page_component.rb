# typed: strict
# frozen_string_literal: true

module Cards
  class PageComponent < ApplicationComponent
    sig { params(page_entity: ::PageEntity, show_topic_name: T::Boolean, class_name: String).void }
    def initialize(page_entity:, show_topic_name: true, class_name: "")
      @page_entity = page_entity
      @show_topic_name = show_topic_name
      @class_name = class_name
    end

    sig { returns(PageEntity) }
    attr_reader :page_entity
    private :page_entity

    sig { returns(T::Boolean) }
    attr_reader :show_topic_name
    private :show_topic_name
    alias_method :show_topic_name?, :show_topic_name

    sig { returns(String) }
    attr_reader :class_name
    private :class_name

    delegate :space_entity, :topic_entity, to: :page_entity
  end
end
