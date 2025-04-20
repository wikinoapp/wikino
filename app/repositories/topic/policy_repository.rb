# typed: strict
# frozen_string_literal: true

class Topic
  class PolicyRepository < ApplicationRepository
    sig { params(user: User, topic: Topic).returns(Topic::Policy) }
    def build(user:, topic:)
      user_record = UserRecord.find(user.database_id)
      topic_record = TopicRecord.find(topic.database_id)

      Topic::Policy.new(
        can_update: user_record.can_update_topic?(topic_record:),
        can_destroy: user_record.can_destroy_topic?(topic_record:)
      )
    end
  end
end
