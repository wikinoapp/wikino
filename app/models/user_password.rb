# typed: strict
# frozen_string_literal: true

class UserPassword < ApplicationRecord
  extend T::Sig

  PASSWORD_MIN_LENGTH = 8

  has_secure_password

  belongs_to :user
end
