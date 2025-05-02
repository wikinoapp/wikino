# typed: strict
# frozen_string_literal: true

module TopicService
  class Create < ApplicationService
    class Result < T::Struct
      const :topic_record, TopicRecord
    end

    sig do
      params(
        space_member_record: SpaceMemberRecord,
        name: String,
        description: String,
        visibility: String
      ).returns(Result)
    end
    def call(space_member_record:, name:, description:, visibility:)
      topic_record = ActiveRecord::Base.transaction do
        new_topic_record = space_member_record.space_record.not_nil!.topic_records.where(name:).first_or_create!(
          description:,
          visibility:
        )
        new_topic_record.add_member!(member: space_member_record, role: TopicMemberRole::Admin)
        new_topic_record
      end

      Result.new(topic_record:)
    end
  end
end
