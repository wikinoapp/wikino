# typed: strict
# frozen_string_literal: true

class TopicRepository < ApplicationRepository
  sig { params(topic_record: TopicRecord).returns(Topic) }
  def to_model(topic_record:)
    Topic.new(
      database_id: topic_record.id,
      space: SpaceRepository.new.to_model(space_record: topic_record.space_record)
    )
  end
end
