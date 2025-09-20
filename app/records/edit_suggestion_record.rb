# typed: strict
# frozen_string_literal: true

class EditSuggestionRecord < ApplicationRecord
  self.table_name = "edit_suggestions"

  enum :status, {
    EditSuggestionStatus::Draft.serialize => 0,
    EditSuggestionStatus::Open.serialize => 1,
    EditSuggestionStatus::Applied.serialize => 2,
    EditSuggestionStatus::Closed.serialize => 3
  }, prefix: true

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :topic_record, foreign_key: :topic_id
  belongs_to :created_user_record, class_name: "UserRecord", foreign_key: :created_user_id
  has_many :edit_suggestion_page_records, foreign_key: :edit_suggestion_id, dependent: :restrict_with_exception
  has_many :comment_records, class_name: "EditSuggestionCommentRecord", foreign_key: :edit_suggestion_id, dependent: :restrict_with_exception

  scope :by_status, ->(status) { where(status:) }
  scope :open_or_draft, -> { where(status: [EditSuggestionStatus::Draft.serialize, EditSuggestionStatus::Open.serialize]) }
  scope :closed_or_applied, -> { where(status: [EditSuggestionStatus::Closed.serialize, EditSuggestionStatus::Applied.serialize]) }

  sig { returns(T::Boolean) }
  def editable?
    status_draft? || status_open?
  end
end
