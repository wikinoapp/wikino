# typed: strict
# frozen_string_literal: true

class UserPasswordRecord < ApplicationRecord
  extend T::Sig

  PASSWORD_MIN_LENGTH = 8

  self.table_name = "user_passwords"

  has_secure_password

  belongs_to :user
end
