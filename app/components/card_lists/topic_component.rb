# typed: strict
# frozen_string_literal: true

module CardLists
  class TopicComponent < ApplicationComponent
    sig { params(topics: T::Array[Topic], current_user_record: T.nilable(UserRecord)).void }
    def initialize(topics:, current_user_record:)
      @topics = topics
      @current_user_record = current_user_record
    end

    sig { returns(T::Array[Topic]) }
    attr_reader :topics
    private :topics

    sig { returns(T.nilable(UserRecord)) }
    attr_reader :current_user_record
    private :current_user_record
  end
end
