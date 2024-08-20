# typed: strict
# frozen_string_literal: true

class ListMember < ApplicationRecord
  extend T::Sig

  belongs_to :space
  belongs_to :list
  belongs_to :user

  enum :role, {
    ListMemberRole::Admin.serialize => 0,
    ListMemberRole::Member.serialize => 1
  }, prefix: true
end
