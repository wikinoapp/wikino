# typed: strict
# frozen_string_literal: true

class DestroyTopicService < ApplicationService
  sig { params(topic_record_id: T::Wikino::DatabaseId).void }
  def call(topic_record_id:)
    topic_record = TopicRecord.find(topic_record_id)

    topic_record.page_records.find_each do |page_record|
      page_record.page_editor_records.destroy_all
      page_record.revision_records.destroy_all
    end

    topic_record.page_records.destroy_all
    topic_record.member_records.destroy_all

    topic_record.destroy!

    nil
  end
end
