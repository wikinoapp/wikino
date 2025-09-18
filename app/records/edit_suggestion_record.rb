# typed: strict
# frozen_string_literal: true

class EditSuggestionRecord < ApplicationRecord
  self.table_name = "edit_suggestions"

  belongs_to :space, class_name: "SpaceRecord"
  belongs_to :topic, class_name: "TopicRecord"
  belongs_to :created_user, class_name: "UserRecord"
  has_many :edit_suggestion_pages, class_name: "EditSuggestionPageRecord", dependent: :destroy
  has_many :comments, class_name: "EditSuggestionCommentRecord", dependent: :destroy

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
