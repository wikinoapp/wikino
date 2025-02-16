# typed: strict
# frozen_string_literal: true

class UpdateSpaceUseCase < ApplicationUseCase
  class Result < T::Struct
    const :space, Space
  end

  sig { params(form: EditSpaceForm).returns(Result) }
  def call(form:)
    space = form.space.not_nil!

    space.attributes = {
      identifier: form.identifier.not_nil!,
      name: form.name.not_nil!
    }
    space.save!

    Result.new(space:)
  end
end
