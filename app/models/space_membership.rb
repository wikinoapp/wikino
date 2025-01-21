# typed: strict
# frozen_string_literal: true

class TopicMembership < ApplicationRecord
  belongs_to :space
  belongs_to :member, class_name: "User"

  enum :role, {
    SpaceMemberRole::Owner.serialize => 0
  }, prefix: true
end
