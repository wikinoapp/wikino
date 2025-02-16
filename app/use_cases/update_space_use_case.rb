# typed: strict
# frozen_string_literal: true

class UpdateSpaceUseCase < ApplicationUseCase
  class Result < T::Struct
    const :space, Space
  end

  sig { params(space: Space, form: EditSpaceForm).returns(Result) }
  def call(space:, form:)
    space.attributes = {
      identifier: form.identifier.not_nil!,
      name: form.name.not_nil!
    }
    space.save!

    Result.new(space:)
  end
end
