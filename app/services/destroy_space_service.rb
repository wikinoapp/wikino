# typed: strict
# frozen_string_literal: true

class DestroySpaceService < ApplicationService
  sig { params(space_record_id: T::Wikino::DatabaseId).void }
  def call(space_record_id:)
    space_record = SpaceRecord.find(space_record_id)

    space_record.topic_records.find_each do |topic_record|
      DestroyTopicService.new.call(topic_record_id: topic_record.id)
    end

    space_record.export_records.destroy_all
    space_record.space_member_records.destroy_all

    space_record.destroy!

    nil
  end
end
