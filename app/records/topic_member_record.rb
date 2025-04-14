# typed: strict
# frozen_string_literal: true

class TopicMemberRecord < ApplicationRecord
  self.table_name = "topic_members"

  belongs_to :space
  belongs_to :topic
  belongs_to :space_member

  enum :role, {
    TopicMemberRole::Admin.serialize => 0,
    TopicMemberRole::Member.serialize => 1
  }, prefix: true
end
