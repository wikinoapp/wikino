# typed: strict
# frozen_string_literal: true

class EditSuggestion < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, Types::DatabaseId
  const :title, String
  const :description, T.nilable(String)
  const :status, EditSuggestionStatus
  const :applied_at, T.nilable(ActiveSupport::TimeWithZone)
  const :created_at, ActiveSupport::TimeWithZone
  const :updated_at, ActiveSupport::TimeWithZone
  const :space, Space
  const :topic, Topic
  const :created_user, User

  sig { returns(T::Boolean) }
  def draft?
    status == EditSuggestionStatus::Draft
  end

  sig { returns(T::Boolean) }
  def open?
    status == EditSuggestionStatus::Open
  end

  sig { returns(T::Boolean) }
  def applied?
    status == EditSuggestionStatus::Applied
  end

  sig { returns(T::Boolean) }
  def closed?
    status == EditSuggestionStatus::Closed
  end

  sig { returns(T::Boolean) }
  def editable?
    draft? || open?
  end

  sig { returns(T::Boolean) }
  def open_or_draft?
    draft? || open?
  end

  sig { returns(T::Boolean) }
  def closed_or_applied?
    closed? || applied?
  end
end
