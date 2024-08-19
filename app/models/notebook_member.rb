# typed: strict
# frozen_string_literal: true

class NotebookMember < ApplicationRecord
  extend T::Sig

  belongs_to :space
  belongs_to :notebook
  belongs_to :user

  enum :role, {
    NotebookMemberRole::Admin.serialize => 0,
    NotebookMemberRole::Member.serialize => 1
  }, prefix: true
end
