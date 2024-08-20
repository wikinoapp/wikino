# typed: strict
# frozen_string_literal: true

class ListPolicy < ApplicationPolicy
  extend T::Sig

  sig { returns(T::Boolean) }
  def show?
    viewer.viewable_lists.where(id: list.id).exists?
  end

  sig { returns(T::Boolean) }
  def update?
    viewer.can_update_list?(list:)
  end

  sig { returns(T::Boolean) }
  def destroy?
    viewer.can_destroy_list?(list:)
  end

  sig { returns(List) }
  private def list
    T.cast(record, List)
  end
end
