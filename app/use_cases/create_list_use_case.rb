# typed: strict
# frozen_string_literal: true

class CreateListUseCase < ApplicationUseCase
  class Result < T::Struct
    const :list, List
  end

  sig { params(viewer: User, name: String, description: String, visibility: String).returns(Result) }
  def call(viewer:, name:, description:, visibility:)
    list = ActiveRecord::Base.transaction do
      list = viewer.space.not_nil!.lists.create!(name:, description:, visibility:)
      list.add_member!(member: viewer, role: ListMemberRole::Admin)
      list
    end

    Result.new(list:)
  end
end
