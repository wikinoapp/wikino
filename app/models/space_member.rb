# typed: strict
# frozen_string_literal: true

class SpaceMember < ApplicationRecord
  enum :role, {
    SpaceMemberRole::Owner.serialize => 0
  }, prefix: true

  belongs_to :space
  belongs_to :user
  has_many :topic_memberships, dependent: :restrict_with_exception, foreign_key: :member_id, inverse_of: :member

  scope :active, -> { where(active: true) }
  scope :inactive, -> { where(active: false) }
end
