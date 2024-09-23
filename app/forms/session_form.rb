# typed: strict
# frozen_string_literal: true

class SessionForm < ApplicationForm
  attribute :space_identifier, :string
  attribute :email, :string
  attribute :password, :string

  validates :space_identifier, presence: true
  validates :email, email: true, presence: true
  validates :password, presence: true
  validate :authentication

  sig { returns(T.nilable(Space)) }
  def space
    @space ||= T.let(Space.find_by(identifier: space_identifier), T.nilable(Space))
  end

  sig { returns(T.nilable(User)) }
  def user
    @user ||= T.let(space&.users&.find_by(email:), T.nilable(User))
  end

  sig { void }
  private def authentication
    unless user&.user_password&.authenticate(password)
      errors.add(:base, :unauthenticated)
    end
  end
end
