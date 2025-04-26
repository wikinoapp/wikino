# typed: strict
# frozen_string_literal: true

class TopicMemberRecord < ApplicationRecord
  self.table_name = "topic_members"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :topic_record, foreign_key: :topic_id
  belongs_to :space_member_record, foreign_key: :space_member_id

  enum :role, {
    TopicMemberRole::Admin.serialize => 0,
    TopicMemberRole::Member.serialize => 1
  }, prefix: true
end
