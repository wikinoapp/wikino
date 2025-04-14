# typed: strict
# frozen_string_literal: true

class UserPasswordRecord < ApplicationRecord
  self.table_name = "user_passwords"

  has_secure_password

  belongs_to :user_record, foreign_key: :user_id
end
