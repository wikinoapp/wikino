# typed: strict
# frozen_string_literal: true

module Dropdowns
  class TopicOptionsComponent < ApplicationComponent
    sig { params(signed_in: T::Boolean, topic: Topic).void }
    def initialize(signed_in:, topic:)
      @signed_in = signed_in
      @topic = topic
    end

    sig { returns(T::Boolean) }
    attr_reader :signed_in
    private :signed_in
    alias_method :signed_in?, :signed_in

    sig { returns(Topic) }
    attr_reader :topic
    private :topic

    delegate :space, to: :topic

    sig { returns(T::Boolean) }
    private def render?
      signed_in?
    end
  end
end
