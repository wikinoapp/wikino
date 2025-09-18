# typed: strict
# frozen_string_literal: true

class EditSuggestionCommentRecord < ApplicationRecord
  self.table_name = "edit_suggestion_comments"

  belongs_to :space, class_name: "SpaceRecord"
  belongs_to :edit_suggestion, class_name: "EditSuggestionRecord"
  belongs_to :created_user, class_name: "UserRecord"

  validates :body, presence: true
end
