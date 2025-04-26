# typed: strict
# frozen_string_literal: true

class UpdateSpaceService < ApplicationService
  class Result < T::Struct
    const :space, SpaceRecord
  end

  sig { params(space: SpaceRecord, form: EditSpaceForm).returns(Result) }
  def call(space:, form:)
    space.attributes = {
      identifier: form.identifier.not_nil!,
      name: form.name.not_nil!
    }
    space.save!

    Result.new(space:)
  end
end
