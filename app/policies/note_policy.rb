# typed: strict
# frozen_string_literal: true

class NotePolicy < ApplicationPolicy
  sig { returns(T::Boolean) }
  def show?
    viewer.role_owner?
  end

  sig { returns(T::Boolean) }
  def create?
    viewer.role_owner?
  end

  sig { returns(T::Boolean) }
  def update?
    viewer.role_owner?
  end

  sig { returns(T::Boolean) }
  def destroy?
    # T.cast(record, Note).user == user
  end
end
