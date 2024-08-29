# typed: strict
# frozen_string_literal: true

class ListMembership < ApplicationRecord
  belongs_to :space
  belongs_to :list
  belongs_to :member, class_name: "User"

  enum :role, {
    ListMemberRole::Admin.serialize => 0,
    ListMemberRole::Member.serialize => 1
  }, prefix: true
end
