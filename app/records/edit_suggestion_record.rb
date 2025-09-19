# typed: strict
# frozen_string_literal: true

class EditSuggestionRecord < ApplicationRecord
  self.table_name = "edit_suggestions"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :topic_record, foreign_key: :topic_id
  belongs_to :created_user_record, foreign_key: :created_user_id
  has_many :edit_suggestion_page_records, foreign_key: :edit_suggestion_id, dependent: :destroy
  has_many :comment_records, class_name: "EditSuggestionCommentRecord", foreign_key: :edit_suggestion_id, dependent: :destroy

  validates :title, presence: true
  validates :status, inclusion: {in: EditSuggestionStatus.values.map(&:serialize)}

  scope :by_status, ->(status) { where(status:) }
  scope :open_or_draft, -> { where(status: [EditSuggestionStatus::Draft.serialize, EditSuggestionStatus::Open.serialize]) }
  scope :closed_or_applied, -> { where(status: [EditSuggestionStatus::Closed.serialize, EditSuggestionStatus::Applied.serialize]) }

  sig { returns(T::Boolean) }
  def draft?
    status == EditSuggestionStatus::Draft.serialize
  end

  sig { returns(T::Boolean) }
  def open?
    status == EditSuggestionStatus::Open.serialize
  end

  sig { returns(T::Boolean) }
  def applied?
    status == EditSuggestionStatus::Applied.serialize
  end

  sig { returns(T::Boolean) }
  def closed?
    status == EditSuggestionStatus::Closed.serialize
  end

  sig { returns(T::Boolean) }
  def editable?
    draft? || open?
  end
end
