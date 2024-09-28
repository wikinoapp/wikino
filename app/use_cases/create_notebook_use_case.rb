# typed: strict
# frozen_string_literal: true

class CreateNotebookUseCase < ApplicationUseCase
  class Result < T::Struct
    const :notebook, Notebook
  end

  sig { params(viewer: User, name: String, description: String, visibility: String).returns(Result) }
  def call(viewer:, name:, description:, visibility:)
    notebook = ActiveRecord::Base.transaction do
      new_notebook = viewer.space.not_nil!.notebooks.create!(name:, description:, visibility:)
      new_notebook.add_member!(member: viewer, role: NotebookMemberRole::Admin)
      new_notebook
    end

    Result.new(notebook:)
  end
end
