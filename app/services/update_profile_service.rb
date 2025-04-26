# typed: strict
# frozen_string_literal: true

class UpdateProfileService < ApplicationService
  class Result < T::Struct
    const :user_record, UserRecord
  end

  sig { params(user_record: UserRecord, atname: String, name: String, description: String).returns(Result) }
  def call(user_record:, atname:, name:, description:)
    user_record.attributes = {atname:, name:, description:}
    user_record.save!

    Result.new(user_record:)
  end
end
