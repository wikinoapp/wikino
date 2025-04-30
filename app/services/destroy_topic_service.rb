# typed: strict
# frozen_string_literal: true

class DestroyTopicService < ApplicationService
  sig { params(topic_record: TopicRecord).void }
  def call(topic_record:)
    topic_record.discard!

    nil
  end
end
