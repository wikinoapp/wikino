# typed: strict
# frozen_string_literal: true

class EditSuggestionPageRevisionRecord < ApplicationRecord
  self.table_name = "edit_suggestion_page_revisions"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :edit_suggestion_page_record, foreign_key: :edit_suggestion_page_id
  belongs_to :editor_space_member_record, foreign_key: :editor_space_member_id, class_name: "SpaceMemberRecord"
end
