# typed: strict
# frozen_string_literal: true

require "securerandom"

module TwoFactorAuthService
  class Enable < ApplicationService
    class Result < T::Struct
      const :recovery_codes, T::Array[String]
    end

    sig { params(user_record: UserRecord, password: String, totp_code: String).returns(Result) }
    def call(user_record:, password:, totp_code:)
      auth_record = user_record.user_two_factor_auth_record.not_nil!
      recovery_codes = generate_recovery_codes

      # 2FAを有効化し、リカバリーコードを保存する
      auth_record.update!(
        enabled: true,
        enabled_at: Time.current,
        recovery_codes:
      )

      Result.new(
        recovery_codes:
      )
    end

    sig { returns(T::Array[String]) }
    private def generate_recovery_codes
      # 10個のリカバリーコードを生成する
      10.times.map do
        # ランダムな8文字の英数字コードを生成する
        SecureRandom.alphanumeric(8).downcase
      end
    end
  end
end
