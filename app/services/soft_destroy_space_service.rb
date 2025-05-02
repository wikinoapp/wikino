# typed: strict
# frozen_string_literal: true

class SoftDestroySpaceService < ApplicationService
  sig { params(space_record: SpaceRecord).void }
  def call(space_record:)
    space_record.topic_records.find_each do |topic_record|
      SoftDestroyTopicService.new.call(topic_record:)
    end

    space_record.discard!

    nil
  end
end
