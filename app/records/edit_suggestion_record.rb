# typed: strict
# frozen_string_literal: true

class EditSuggestionRecord < ApplicationRecord
  self.table_name = "edit_suggestions"

  STATUSES = T.let(%w[draft open applied closed].freeze, T::Array[String])

  belongs_to :space, class_name: "SpaceRecord"
  belongs_to :topic, class_name: "TopicRecord"
  belongs_to :created_user, class_name: "UserRecord"
  has_many :edit_suggestion_pages, class_name: "EditSuggestionPageRecord", dependent: :destroy
  has_many :comments, class_name: "EditSuggestionCommentRecord", dependent: :destroy

  validates :title, presence: true
  validates :status, inclusion: {in: STATUSES}

  scope :by_status, ->(status) { where(status:) }
  scope :open_or_draft, -> { where(status: %w[draft open]) }
  scope :closed_or_applied, -> { where(status: %w[closed applied]) }

  sig { returns(T::Boolean) }
  def draft?
    status == "draft"
  end

  sig { returns(T::Boolean) }
  def open?
    status == "open"
  end

  sig { returns(T::Boolean) }
  def applied?
    status == "applied"
  end

  sig { returns(T::Boolean) }
  def closed?
    status == "closed"
  end

  sig { returns(T::Boolean) }
  def editable?
    draft? || open?
  end
end
