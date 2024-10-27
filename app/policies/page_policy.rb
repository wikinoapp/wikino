# typed: strict
# frozen_string_literal: true

class PagePolicy < ApplicationPolicy
  sig { returns(T::Boolean) }
  def show?
    user.role_owner?
  end

  sig { returns(T::Boolean) }
  def create?
    user.role_owner?
  end

  sig { returns(T::Boolean) }
  def update?
    user.role_owner?
  end

  sig { returns(T::Boolean) }
  def destroy?
    # T.cast(record, Page).user == user
    false
  end
end
