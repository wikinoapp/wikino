# typed: strict
# frozen_string_literal: true

class BulkRestoredPagePolicy < ApplicationPolicy
  include PolicyConcerns::SpaceContext

  sig { returns(T::Boolean) }
  def create?
    return false unless same_space_member?

    space_member_record!.active?
  end
end
