# typed: strict
# frozen_string_literal: true

class CreateTopicService < ApplicationService
  class Result < T::Struct
    const :topic, TopicRecord
  end

  sig { params(space_member: SpaceMemberRecord, name: String, description: String, visibility: String).returns(Result) }
  def call(space_member:, name:, description:, visibility:)
    topic = ActiveRecord::Base.transaction do
      new_topic = space_member.space.not_nil!.topics.where(name:).first_or_create!(description:, visibility:)
      new_topic.add_member!(member: space_member, role: TopicMemberRole::Admin)
      new_topic
    end

    Result.new(topic:)
  end
end
