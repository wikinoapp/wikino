# typed: strict
# frozen_string_literal: true

module TwoFactorAuthService
  class ConsumeRecoveryCode < ApplicationService
    sig { params(user_record: UserRecord, recovery_code: String).returns(T::Boolean) }
    def call(user_record:, recovery_code:)
      auth_record = user_record.user_two_factor_auth_record
      return false if auth_record.nil?

      ActiveRecord::Base.transaction do
        # リカバリーコードが存在するか確認
        return false unless auth_record.recovery_codes.include?(recovery_code)

        # リカバリーコードを削除
        auth_record.recovery_codes.delete(recovery_code)
        auth_record.save!

        true
      end
    end
  end
end