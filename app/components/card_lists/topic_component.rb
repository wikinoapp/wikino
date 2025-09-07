# typed: strict
# frozen_string_literal: true

module CardLists
  class TopicComponent < ApplicationComponent
    sig { params(topics: T::Array[Topic], is_guest: T::Boolean).void }
    def initialize(topics:, is_guest:)
      @topics = topics
      @is_guest = is_guest
    end

    sig { returns(T::Array[Topic]) }
    attr_reader :topics
    private :topics

    sig { returns(T::Boolean) }
    attr_reader :is_guest
    private :is_guest
  end
end
