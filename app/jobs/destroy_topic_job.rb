# typed: strict
# frozen_string_literal: true

class DestroyTopicJob < ApplicationJob
  queue_as :low

  sig { params(topic_record_id: Types::DatabaseId).void }
  def perform(topic_record_id:)
    Topics::DestroyService.new.call(topic_record_id:)
  end
end
