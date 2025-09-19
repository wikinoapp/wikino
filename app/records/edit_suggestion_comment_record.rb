# typed: strict
# frozen_string_literal: true

class EditSuggestionCommentRecord < ApplicationRecord
  self.table_name = "edit_suggestion_comments"

  belongs_to :space_record, class_name: "SpaceRecord", foreign_key: :space_id
  belongs_to :edit_suggestion_record, class_name: "EditSuggestionRecord", foreign_key: :edit_suggestion_id
  belongs_to :created_user_record, class_name: "UserRecord", foreign_key: :created_user_id

  validates :body, presence: true
end
