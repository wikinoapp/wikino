# typed: strict
# frozen_string_literal: true

class SpacePolicy < ApplicationPolicy
  sig { returns(T::Boolean) }
  def show?
    user.space == record
  end
end
