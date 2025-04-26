# typed: strict
# frozen_string_literal: true

class ConfirmEmailService < ApplicationService
  class Result < T::Struct
    const :email_confirmation_record, EmailConfirmationRecord
  end

  sig { params(email_confirmation_record: EmailConfirmationRecord, user_record: T.nilable(UserRecord)).returns(Result) }
  def call(email_confirmation_record:, user_record: nil)
    ActiveRecord::Base.transaction do
      email_confirmation_record.success!

      user_record&.run_after_email_confirmation_success!(email_confirmation_record:)
    end

    Result.new(email_confirmation_record:)
  end
end
