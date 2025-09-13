# typed: strict
# frozen_string_literal: true

module JoinedTopics
  class IndexView < ApplicationView
    sig { params(topics: T::Array[Topic], variant: Symbol).void }
    def initialize(topics:, variant:)
      @topics = topics
      @variant = variant
    end

    sig { returns(T::Array[Topic]) }
    attr_reader :topics
    private :topics

    sig { returns(Symbol) }
    attr_reader :variant
    private :variant

    sig { returns(String) }
    private def turbo_frame_id
      "joined-topics-#{variant}"
    end
  end
end
