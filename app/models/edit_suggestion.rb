# typed: strict
# frozen_string_literal: true

class EditSuggestion
  extend T::Sig

  sig { returns(Types::DatabaseId) }
  attr_reader :id

  sig { returns(Types::DatabaseId) }
  attr_reader :space_id

  sig { returns(Types::DatabaseId) }
  attr_reader :topic_id

  sig { returns(Types::DatabaseId) }
  attr_reader :created_user_id

  sig { returns(String) }
  attr_reader :title

  sig { returns(T.nilable(String)) }
  attr_reader :description

  sig { returns(String) }
  attr_reader :status

  sig { returns(T.nilable(ActiveSupport::TimeWithZone)) }
  attr_reader :applied_at

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :created_at

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :updated_at

  sig do
    params(
      id: Types::DatabaseId,
      space_id: Types::DatabaseId,
      topic_id: Types::DatabaseId,
      created_user_id: Types::DatabaseId,
      title: String,
      description: T.nilable(String),
      status: String,
      applied_at: T.nilable(ActiveSupport::TimeWithZone),
      created_at: ActiveSupport::TimeWithZone,
      updated_at: ActiveSupport::TimeWithZone
    ).void
  end
  def initialize(
    id:,
    space_id:,
    topic_id:,
    created_user_id:,
    title:,
    description:,
    status:,
    applied_at:,
    created_at:,
    updated_at:
  )
    @id = id
    @space_id = space_id
    @topic_id = topic_id
    @created_user_id = created_user_id
    @title = title
    @description = description
    @status = status
    @applied_at = applied_at
    @created_at = created_at
    @updated_at = updated_at
  end

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

  sig { returns(T::Boolean) }
  def open_or_draft?
    draft? || open?
  end

  sig { returns(T::Boolean) }
  def closed_or_applied?
    closed? || applied?
  end
end
