# typed: strict
# frozen_string_literal: true

class UpdateProfileService < ApplicationService
  class Result < T::Struct
    const :user, UserRecord
  end

  sig { params(form: EditProfileForm).returns(Result) }
  def call(form:)
    user = form.user.not_nil!
    user.attributes = {
      atname: form.atname.not_nil!,
      name: form.name.not_nil!,
      description: form.description.not_nil!
    }

    user.save!

    Result.new(user:)
  end
end
