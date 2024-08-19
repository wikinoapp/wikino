# typed: strict
# frozen_string_literal: true

class NotebookPolicy < ApplicationPolicy
  extend T::Sig

  sig { returns(T::Boolean) }
  def show?
    viewer.viewable_notebooks.where(id: notebook.id).exists?
  end

  sig { returns(T::Boolean) }
  def update?
    viewer.can_update_notebook?(notebook:)
  end

  sig { returns(T::Boolean) }
  def destroy?
    viewer.can_destroy_notebook?(notebook:)
  end

  sig { returns(Notebook) }
  private def notebook
    T.cast(record, Notebook)
  end
end
