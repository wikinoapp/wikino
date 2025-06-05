# typed: strict
# frozen_string_literal: true

module UserSessionService
  class CreateWithRecoveryCode < ApplicationService
    class Result < T::Struct
      const :user_session_record, UserSessionRecord
    end

    sig do
      params(
        user_two_factor_auth_record: UserTwoFactorAuthRecord,
        recovery_code: String,
        ip_address: T.nilable(String),
        user_agent: T.nilable(String)
      ).returns(Result)
    end
    def call(user_two_factor_auth_record:, recovery_code:, ip_address:, user_agent:)
      user_record = user_two_factor_auth_record.user_record.not_nil!

      user_session_record = ActiveRecord::Base.transaction do
        # リカバリーコードを消費
        user_two_factor_auth_record.consume_recovery_code(recovery_code:)

        # セッションを作成
        user_record.user_session_records.start!(ip_address:, user_agent:)
      end

      Result.new(user_session_record:)
    end
  end
end
