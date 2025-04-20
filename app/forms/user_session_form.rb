# typed: strict
# frozen_string_literal: true

class UserSessionForm < ApplicationForm
  attribute :email, :string
  attribute :password, :string

  validates :email, email: true, presence: true
  validates :password, presence: true
  validate :authentication

  sig { returns(T.nilable(User)) }
  def user
    @user ||= T.let(User.kept.find_by(email:), T.nilable(User))
  end

  sig { void }
  private def authentication
    unless user&.user_password&.authenticate(password)
      errors.add(:base, :unauthenticated)
    end
  end
end
