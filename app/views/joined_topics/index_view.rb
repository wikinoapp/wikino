# typed: strict
# frozen_string_literal: true

module JoinedTopics
  class IndexView < ApplicationView
    sig { params(topics: T::Array[Topic]).void }
    def initialize(topics:)
      @topics = topics
    end

    sig { returns(T::Array[Topic]) }
    attr_reader :topics
    private :topics
  end
end
