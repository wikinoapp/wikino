# typed: strict
# frozen_string_literal: true

class ExportService < ApplicationService
  class Result < T::Struct
    const :export, Export
  end

  sig { params(space: Space, started_by: SpaceMember).returns(Result) }
  def call(space:, started_by:)
    export = space.exports.create!(
      started_by:,
      started_at: Time.current
    )

    Result.new(export:)
  end
end
