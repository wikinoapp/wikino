# typed: strict
# frozen_string_literal: true

class TopicMembership < ApplicationRecord
  belongs_to :space
  belongs_to :topic
  belongs_to :member, class_name: "User"

  enum :role, {
    TopicMemberRole::Admin.serialize => 0,
    TopicMemberRole::Member.serialize => 1
  }, prefix: true
end
