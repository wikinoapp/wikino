# typed: strict
# frozen_string_literal: true

class UpdateTopicService < ApplicationService
  class Result < T::Struct
    const :topic_record, TopicRecord
  end

  sig do
    params(
      topic_record: TopicRecord,
      name: String,
      description: String,
      visibility: String
    ).returns(Result)
  end
  def call(topic_record:, name:, description:, visibility:)
    topic_record.attributes = {name:, description:, visibility:}
    topic_record.save!

    Result.new(topic_record:)
  end
end
