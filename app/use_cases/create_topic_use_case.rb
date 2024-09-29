# typed: strict
# frozen_string_literal: true

class CreateTopicUseCase < ApplicationUseCase
  class Result < T::Struct
    const :topic, Topic
  end

  sig { params(viewer: User, name: String, description: String, visibility: String).returns(Result) }
  def call(viewer:, name:, description:, visibility:)
    topic = ActiveRecord::Base.transaction do
      new_topic = viewer.space.not_nil!.topics.create!(name:, description:, visibility:)
      new_topic.add_member!(member: viewer, role: TopicMemberRole::Admin)
      new_topic
    end

    Result.new(topic:)
  end
end
