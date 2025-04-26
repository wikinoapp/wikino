# typed: strict
# frozen_string_literal: true

class ConfirmEmailService < ApplicationService
  class Result < T::Struct
    const :email_confirmation_record, EmailConfirmationRecord
  end

  sig { params(user_record: UserRecord, email_confirmation_record: EmailConfirmationRecord).returns(Result) }
  def call(user_record:, email_confirmation_record:)
    ActiveRecord::Base.transaction do
      email_confirmation_record.success!

      user_record.run_after_email_confirmation_success!(email_confirmation_record:)
    end

    Result.new(email_confirmation_record:)
  end
end
