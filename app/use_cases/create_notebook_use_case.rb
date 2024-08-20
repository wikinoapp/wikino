# typed: strict
# frozen_string_literal: true

class CreateNotebookUseCase < ApplicationUseCase
  class Result < T::Struct
    const :notebook, Notebook
  end

  sig do
    params(viewer: User, identifier: String, visibility: String, name: String, description: String)
      .returns(Result)
  end
  def call(viewer:, identifier:, visibility:, name:, description:)
    notebook = ActiveRecord::Base.transaction do
      viewer.space.notebooks.create!(identifier:, visibility:, name:, description:)
    end

    Result.new(notebook:)
  end
end
