# typed: strict
# frozen_string_literal: true

class UserTwoFactorAuthRecord < ApplicationRecord
  self.table_name = "user_two_factor_auths"

  belongs_to :user_record, foreign_key: :user_id

  sig { params(code: String).returns(T::Boolean) }
  def recovery_code_valid?(code)
    return false unless recovery_codes.include?(code)

    # リカバリーコードは一度しか使えないため、使用後は削除
    update!(recovery_codes: recovery_codes - [code])
    true
  end
end
