# typed: strict
# frozen_string_literal: true

class CreateListUseCase < ApplicationUseCase
  class Result < T::Struct
    const :list, List
  end

  sig do
    params(viewer: User, identifier: String, visibility: String, name: String, description: String)
      .returns(Result)
  end
  def call(viewer:, identifier:, visibility:, name:, description:)
    list = ActiveRecord::Base.transaction do
      viewer.space.lists.create!(identifier:, visibility:, name:, description:)
    end

    Result.new(list:)
  end
end
