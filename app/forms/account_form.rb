# typed: strict
# frozen_string_literal: true

class AccountForm < ApplicationForm
  include FormConcerns::PasswordValidatable

  attribute :email, :string
  attribute :locale, :string
  attribute :password, :string
  attribute :time_zone, :string

  validates :atname,
    format: {with: User::ATNAME_FORMAT},
    length: {in: User::ATNAME_MIN_LENGTH..User::ATNAME_MAX_LENGTH},
    presence: true,
    unreserved_atname: true
  validates :email, email: true, presence: true
  validates :locale, presence: true
  validates :time_zone, presence: true
  validate :atname_uniqueness

  sig { void }
  private def atname_uniqueness
    if Profile.exists?(atname:)
      errors.add(:atname, :uniqueness)
    end
  end
end
