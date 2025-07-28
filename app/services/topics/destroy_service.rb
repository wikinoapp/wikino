# typed: strict
# frozen_string_literal: true

module Topics
  class DestroyService < ApplicationService
    sig { params(topic_record_id: T::Wikino::DatabaseId).void }
    def call(topic_record_id:)
      topic_record = TopicRecord.find(topic_record_id)

      topic_record.page_records.destroy_all_with_related_records!
      topic_record.member_records.destroy_all

      topic_record.destroy!

      nil
    end
  end
end
