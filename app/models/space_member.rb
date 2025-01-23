# typed: strict
# frozen_string_literal: true

class SpaceMember < ApplicationRecord
  belongs_to :space
  belongs_to :user

  enum :role, {
    SpaceMemberRole::Owner.serialize => 0
  }, prefix: true
end
