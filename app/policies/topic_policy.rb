# typed: strict
# frozen_string_literal: true

class TopicPolicy < ApplicationPolicy
  extend T::Sig

  sig { returns(T::Boolean) }
  def show?
    return true if user.nil? && record.visibility_public?
    return false if user.nil?
    user.can_update_topic?(topic:)
  end

  sig { returns(T::Boolean) }
  def create?
    user&.can_update_topic?(topic:) == true
  end

  sig { returns(T::Boolean) }
  def update?
    create?
  end

  sig { returns(T::Boolean) }
  def destroy?
    user.can_destroy_topic?(topic:)
  end

  sig { returns(Topic) }
  private def topic
    T.cast(record, Topic)
  end
end
