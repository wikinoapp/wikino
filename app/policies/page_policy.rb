# typed: strict
# frozen_string_literal: true

class PagePolicy < ApplicationPolicy
  sig { returns(T::Boolean) }
  def show?
    return true if user.nil? && record.topic.visibility_public?
    return false if user.nil?
    user.role_owner?
  end

  sig { returns(T::Boolean) }
  def create?
    user&.role_owner? == true
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
