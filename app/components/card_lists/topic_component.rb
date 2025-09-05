# typed: strict
# frozen_string_literal: true

module CardLists
  class TopicComponent < ApplicationComponent
    sig { params(topics: T::Array[Topic]).void }
    def initialize(topics:)
      @topics = topics
    end

    sig { returns(T::Array[Topic]) }
    attr_reader :topics
    private :topics
  end
end
