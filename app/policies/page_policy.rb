# typed: strict
# frozen_string_literal: true

class PagePolicy < ApplicationPolicy
  sig { returns(T::Boolean) }
  def create?
    user&.role_owner? == true ||
      user&.joined_topic?(topic: T.cast(record, Page).topic.not_nil!) == true
  end

  sig { returns(T::Boolean) }
  def update?
    create?
  end

  sig { returns(T::Boolean) }
  def destroy?
    # T.cast(record, Page).user == user
    false
  end
end
