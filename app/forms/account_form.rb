# typed: strict
# frozen_string_literal: true

class AccountForm < ApplicationForm
  include FormConcerns::PasswordValidatable

  attribute :space_identifier, :string
  attribute :email, :string
  attribute :locale, :string
  attribute :password, :string
  attribute :time_zone, :string

  validates :space_identifier,
    exclusion: {in: Space::RESERVED_IDENTIFIERS},
    format: {with: Space::IDENTIFIER_FORMAT},
    length: {minimum: Space::IDENTIFIER_MIN_LENGTH, maximum: Space::IDENTIFIER_MAX_LENGTH},
    presence: true
  validates :email, email: true, presence: true
  validates :atname, presence: true
  validates :locale, presence: true
  validates :time_zone, presence: true
  validate :space_identifier_uniqueness

  sig { returns(String) }
  def atname
    @atname ||= T.let(SecureRandom.alphanumeric(6), T.nilable(String))
  end

  sig { void }
  private def space_identifier_uniqueness
    if Space.find_by(identifier: space_identifier)
      errors.add(:space_identifier, :uniqueness)
    end
  end
end
