# typed: strict
# frozen_string_literal: true

class Topic
  class PolicyRepository < ApplicationRepository
    sig { params(user_record: UserRecord, topic_record: TopicRecord).returns(Topic::Policy) }
    def build(user_record:, topic_record:)
      Topic.Policy.new(
        can_update: user_record.can_update_topic?(topic_record:),
        can_destroy: user_record.can_destroy_topic?(topic_record:)
      )
    end
  end
end
