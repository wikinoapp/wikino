# typed: strict
# frozen_string_literal: true

module Dropdowns
  class TopicOptionsComponent < ApplicationComponent
    sig { params(signed_in: T::Boolean, topic_entity: TopicEntity).void }
    def initialize(signed_in:, topic_entity:)
      @signed_in = signed_in
      @topic_entity = topic_entity
    end

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in
    alias_method :signed_in?, :signed_in

    sig { returns(TopicEntity) }
    attr_reader :topic_entity
    private :topic_entity

    delegate :space_entity, to: :topic_entity

    sig { returns(T::Boolean) }
    private def render?
      signed_in?
    end
  end
end
