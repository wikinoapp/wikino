# typed: strict
# frozen_string_literal: true

class NotebookMembership < ApplicationRecord
  belongs_to :space
  belongs_to :notebook
  belongs_to :member, class_name: "User"

  enum :role, {
    NotebookMemberRole::Admin.serialize => 0,
    NotebookMemberRole::Member.serialize => 1
  }, prefix: true
end
