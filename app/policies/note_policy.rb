# typed: strict
# frozen_string_literal: true

class NotePolicy < ApplicationPolicy
  extend T::Sig

  sig { returns(T::Boolean) }
  def show?
    T.cast(record, Note).user == user
  end

  sig { returns(T::Boolean) }
  def destroy?
    T.cast(record, Note).user == user
  end

  class Scope < ApplicationPolicy::Scope
    sig { override.returns(ActiveRecord::Relation) }
    def resolve
      user.notes
    end
  end
end
