# typed: strict
# frozen_string_literal: true

class TopicRepository < ApplicationRepository
  sig { params(topic_record: TopicRecord, can_create_page: T.nilable(T::Boolean)).returns(Topic) }
  def to_model(topic_record:, can_create_page: nil)
    Topic.new(
      database_id: topic_record.id,
      number: topic_record.number,
      name: topic_record.name,
      description: topic_record.description,
      visibility: TopicVisibility.deserialize(topic_record.visibility),
      can_create_page:,
      space: SpaceRepository.new.to_model(space_record: topic_record.space_record)
    )
  end
end
