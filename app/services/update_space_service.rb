# typed: strict
# frozen_string_literal: true

class UpdateSpaceService < ApplicationService
  class Result < T::Struct
    const :space_record, SpaceRecord
  end

  sig { params(space_record: SpaceRecord, identifier: String, name: String).returns(Result) }
  def call(space_record:, identifier:, name:)
    space_record.attributes = {identifier:, name:}
    space_record.save!

    Result.new(space_record:)
  end
end
