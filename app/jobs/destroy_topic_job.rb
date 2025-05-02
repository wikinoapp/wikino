# typed: strict
# frozen_string_literal: true

class DestroyTopicJob < ApplicationJob
  queue_as :low

  sig { params(topic_record_id: T::Wikino::DatabaseId).void }
  def perform(topic_record_id:)
    TopicService::Destroy.new.call(topic_record_id:)
  end
end
