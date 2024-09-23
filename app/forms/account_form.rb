# typed: strict
# frozen_string_literal: true

class AccountForm < ApplicationForm
  include FormConcerns::PasswordValidatable

  attribute :email, :string
  attribute :locale, :string
  attribute :password, :string
  attribute :time_zone, :string

  validates :space_identifier, presence: true
  validates :email, email: true, presence: true
  validates :atname, presence: true
  validates :locale, presence: true
  validates :time_zone, presence: true
  validate :space_identifier_uniqueness

  sig { returns(String) }
  def space_identifier
    @space_identifier ||= T.let(SecureRandom.alphanumeric(6), String)
  end

  sig { returns(String) }
  def atname
    @atname ||= T.let(SecureRandom.alphanumeric(6), String)
  end

  sig { void }
  private def space_identifier_uniqueness
    if Space.find_by(identifier: space_identifier)
      errors.add(:space_identifier, :uniqueness)
    end
  end
end
