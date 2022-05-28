# typed: strict
# frozen_string_literal: true

class NotePolicy < ApplicationPolicy
  extend T::Sig

  sig { returns(T::Boolean) }
  def show?
    T.cast(record, Note).user == user
  end
end
