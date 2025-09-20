# typed: strict
# frozen_string_literal: true

class EditSuggestionCommentRecord < ApplicationRecord
  self.table_name = "edit_suggestion_comments"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :edit_suggestion_record, foreign_key: :edit_suggestion_id
  belongs_to :created_space_member_record, class_name: "SpaceMemberRecord", foreign_key: :created_space_member_id
end
