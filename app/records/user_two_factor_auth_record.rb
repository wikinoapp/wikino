# typed: strict
# frozen_string_literal: true

class UserTwoFactorAuthRecord < ApplicationRecord
  self.table_name = "user_two_factor_auths"

  belongs_to :user_record, foreign_key: :user_id

  sig { params(recovery_code: String).returns(T::Boolean) }
  def recovery_code_valid?(recovery_code:)
    recovery_codes.include?(recovery_code)
  end

  sig { params(recovery_code: String).returns(T::Boolean) }
  def consume_recovery_code(recovery_code:)
    recovery_codes.delete(recovery_code)
    save!
  end
end
