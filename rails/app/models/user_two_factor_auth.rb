# typed: strict
# frozen_string_literal: true

require "rotp"

class UserTwoFactorAuth < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, Types::DatabaseId
  const :user_id, Types::DatabaseId
  const :secret, String, sensitivity: []
  const :enabled, T::Boolean
  const :enabled_at, T.nilable(ActiveSupport::TimeWithZone)
  const :recovery_codes, T::Array[String]

  sig { returns(T::Array[String]) }
  def self.generate_recovery_codes
    # 10個のリカバリーコードを生成
    10.times.map do
      # 8文字の英数字小文字でランダム生成
      SecureRandom.alphanumeric(8).downcase
    end
  end

  sig { returns(ROTP::TOTP) }
  def totp
    @totp ||= T.let(ROTP::TOTP.new(secret, issuer: "Wikino"), T.nilable(ROTP::TOTP))
  end

  sig { params(code: String).returns(T::Boolean) }
  def verify_code(code)
    totp.verify(code, drift_behind: 15, drift_ahead: 15).present?
  end

  sig { params(user: User).returns(String) }
  def provisioning_uri(user:)
    totp.provisioning_uri(user.email)
  end

  sig { returns(String) }
  def current_code
    totp.now
  end

  sig { returns(T::Boolean) }
  def enabled?
    enabled
  end

  sig { returns(T::Boolean) }
  def disabled?
    !enabled
  end
end
