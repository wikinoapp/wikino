# typed: strict
# frozen_string_literal: true

module Topics
  class SoftDestroyService < ApplicationService
    sig { params(topic_record: TopicRecord).void }
    def call(topic_record:)
      topic_record.discard!

      DestroyTopicJob.perform_later(topic_record_id: topic_record.id)

      nil
    end
  end
end
