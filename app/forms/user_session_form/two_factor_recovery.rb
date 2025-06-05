# typed: strict
# frozen_string_literal: true

module UserSessionForm
  class TwoFactorRecovery < ApplicationForm
    sig { returns(T.nilable(UserRecord)) }
    attr_accessor :user_record

    attribute :recovery_code, :string

    validates :recovery_code, presence: true, length: {is: 8}, format: {with: /\A[a-z0-9]{8}\z/}
    validate :verify_recovery_code

    sig { void }
    private def verify_recovery_code
      return if recovery_code.blank? || user_record.nil?
      return unless user_record.two_factor_enabled?

      auth_record = user_record.user_two_factor_auth_record.not_nil!

      unless auth_record.recovery_code_valid?(recovery_code:)
        errors.add(:recovery_code, :invalid_code)
      end
    end
  end
end
