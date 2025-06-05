# typed: strict
# frozen_string_literal: true

class UserTwoFactorAuthRecord < ApplicationRecord
  self.table_name = "user_two_factor_auths"

  belongs_to :user_record, foreign_key: :user_id

  sig { params(code: String).returns(T::Boolean) }
  def recovery_code_valid?(code)
    recovery_codes.include?(code)
  end
end
